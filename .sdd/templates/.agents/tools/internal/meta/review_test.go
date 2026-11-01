package meta

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestReviewConcurrentProcess(t *testing.T) {
	root := os.Getenv("REVIEW_TEST_ROOT")
	if root == "" {
		return
	}
	r, err := ReadReview(filepath.Join(root, ReviewRel("src/specs/test/spec.md")))
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("ready")
	if _, err := io.ReadFull(os.Stdin, make([]byte, 1)); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("REVIEW_TEST_CLOSE") == "1" {
		err = CompleteReview(root, r.Source, r.RawSum, func() error {
			return os.WriteFile(filepath.Join(root, "reviewed"), []byte(r.Hash), 0o644)
		})
	} else {
		r.Paths = []string{os.Getenv("REVIEW_TEST_WRITER")}
		err = WriteReviewAtomic(root, r)
	}
	if err != nil {
		if !strings.Contains(err.Error(), "concurrently") {
			t.Fatal(err)
		}
		fmt.Println("conflict")
	} else {
		fmt.Println("accepted")
	}
}

func TestReviewCompareAndChangeAcrossProcesses(t *testing.T) {
	for _, closing := range []bool{false, true} {
		t.Run(fmt.Sprintf("close=%v", closing), func(t *testing.T) {
			root := t.TempDir()
			r := &Review{Version: ReviewVersion, Source: "src/specs/test/spec.md", Hash: "original", Scope: "unknown"}
			if err := WriteReviewAtomic(root, r); err != nil {
				t.Fatal(err)
			}
			type worker struct {
				cmd *exec.Cmd
				in  io.WriteCloser
				out *bufio.Reader
			}
			var workers []worker
			for i := 0; i < 24; i++ {
				cmd := exec.Command(os.Args[0], "-test.run=^TestReviewConcurrentProcess$")
				cmd.Env = append(os.Environ(), "REVIEW_TEST_ROOT="+root, fmt.Sprintf("REVIEW_TEST_WRITER=%d", i))
				if closing && i%2 == 0 {
					cmd.Env = append(cmd.Env, "REVIEW_TEST_CLOSE=1")
				}
				in, err := cmd.StdinPipe()
				if err != nil {
					t.Fatal(err)
				}
				out, err := cmd.StdoutPipe()
				if err != nil {
					t.Fatal(err)
				}
				cmd.Stderr = os.Stderr
				if err := cmd.Start(); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = cmd.Process.Kill() })
				reader := bufio.NewReader(out)
				if line, err := reader.ReadString('\n'); err != nil || line != "ready\n" {
					t.Fatalf("worker not ready: %q, %v", line, err)
				}
				workers = append(workers, worker{cmd, in, reader})
			}
			for _, w := range workers {
				if _, err := w.in.Write([]byte{1}); err != nil {
					t.Fatal(err)
				}
				w.in.Close()
			}
			accepted := 0
			for _, w := range workers {
				out, err := io.ReadAll(w.out)
				if err != nil {
					t.Fatal(err)
				}
				if err := w.cmd.Wait(); err != nil {
					t.Fatalf("worker: %v: %s", err, out)
				}
				if strings.Contains(string(out), "accepted") {
					accepted++
				}
			}
			if accepted != 1 {
				t.Fatalf("accepted %d updates from one revision, want 1", accepted)
			}
		})
	}
}

func TestReviewCloseConflictDoesNotAdvanceReviewed(t *testing.T) {
	root := t.TempDir()
	r := &Review{Version: ReviewVersion, Source: "src/specs/test/spec.md", Hash: "old", Scope: "unknown"}
	if err := WriteReviewAtomic(root, r); err != nil {
		t.Fatal(err)
	}
	stale := r.RawSum
	r.Hash = "new"
	if err := WriteReviewAtomic(root, r); err != nil {
		t.Fatal(err)
	}
	called := false
	err := CompleteReview(root, r.Source, stale, func() error { called = true; return nil })
	if err == nil || called {
		t.Fatalf("stale close: err=%v, saved=%v", err, called)
	}
}
