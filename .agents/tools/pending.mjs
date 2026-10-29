#!/usr/bin/env node
// pending: which tracking entries under perspectives/ are `[ ]`, what holds
// each one back, and the next `/implement continue` batch.
//
// Usage: node .agents/tools/pending.mjs [--batch] [--root <dir>]
//   --batch       print only the next batch and the drift
//   --root <dir>  read <dir>/perspectives instead of this repository's
//
// Output, plain text:
//   # Pending     perspective → unit → `[ ]` rule: layer, state (new, changed,
//                 removed, interrupted), path count, pending cites, parked gap
//   # Next batch  the perspective, its units in order, the rules to run
//                 (Removed: first) and the rules that wait, with the reason
//   # Drift       spec rules with no tracking entry, entries with no rule and
//                 no Removed:, duplicate entries, spec gaps matching no `[ ]` rule
// With no `[ ]` entry it prints "No pending entries." and the drift.
//
// It implements **Continue** of .agents/skills/implement/SKILL.md and reads
// the tracking and TODO shapes of perspectives/AGENTS.md; they win over it.

import { existsSync, readFileSync, readdirSync, statSync } from 'node:fs';
import { dirname, join, relative, resolve, sep } from 'node:path';
import { fileURLToPath } from 'node:url';

const ORDER = ['development', 'business', 'user', 'operator'];
const CAP = 5;
const LAYERS = ['policy', 'mechanism'];
const GLOBAL_SLOTS = { 'global policy rules': 'policy', 'global mechanism rules': 'mechanism' };

function parseArgs(argv) {
  const opts = { batch: false, root: resolve(dirname(fileURLToPath(import.meta.url)), '..', '..') };
  for (let i = 0; i < argv.length; i++) {
    if (argv[i] === '--batch') opts.batch = true;
    else if (argv[i] === '--root' && argv[i + 1]) opts.root = resolve(argv[++i]);
    else fail(`unknown argument: ${argv[i]}\nusage: node .agents/tools/pending.mjs [--batch] [--root <dir>]`, 2);
  }
  return opts;
}

function fail(message, code = 1) {
  process.stderr.write(`pending: ${message}\n`);
  process.exit(code);
}

// Headings of level 1–3 outside code fences, each with its lines up to the next one.
function sections(text) {
  const out = [];
  let fence = null;
  let current = null;
  for (const line of text.split(/\r?\n/)) {
    const marker = line.match(/^\s*(`{3,}|~{3,})/);
    if (marker) fence = !fence ? marker[1][0] : marker[1][0] === fence ? null : fence;
    const heading = !fence && !marker && line.match(/^(#{1,3})\s+(.+?)\s*$/);
    if (heading) out.push((current = { level: heading[1].length, title: heading[2], lines: [] }));
    else if (current) current.lines.push(line);
  }
  return out;
}

// `###` rules of a feature spec file, or of the global slots of a perspective AGENTS.md.
function specRules(file, fileLayer) {
  const rules = [];
  let layer = fileLayer;
  for (const s of sections(readFileSync(file, 'utf8'))) {
    if (s.level < 3 && !fileLayer) layer = GLOBAL_SLOTS[s.title.toLowerCase()] ?? null;
    if (s.level === 3 && layer) rules.push({ title: s.title, layer, body: s.lines.join('\n') });
  }
  return rules;
}

function trackingEntries(file) {
  const entries = [];
  for (const s of sections(readFileSync(file, 'utf8'))) {
    const m = s.level === 3 && s.title.match(/^\[([ xX])\]\s+(.+)$/);
    if (!m) continue;
    const entry = { title: m[2], done: m[1] !== ' ', paths: [], changed: null, removed: null, dependency: [] };
    for (const raw of s.lines) {
      const line = raw.trim().replace(/^-\s+/, '');
      const path = line.match(/^`([^`]+)`$/);
      if (path) entry.paths.push(path[1]);
      else if (line.startsWith('Changed:')) entry.changed = line;
      else if (line.startsWith('Removed:')) entry.removed = line;
      else if (line.startsWith('Dependency:')) entry.dependency.push(line);
    }
    entries.push(entry);
  }
  return entries;
}

function loadTree(root) {
  const base = join(root, 'perspectives');
  if (!existsSync(base)) fail(`no perspectives/ under ${root}`);
  const rel = file => relative(root, file).split(sep).join('/');
  const subdirs = dir => readdirSync(dir).filter(n => !n.startsWith('.') && statSync(join(dir, n)).isDirectory()).sort();
  const found = subdirs(base);
  const perspectives = [...ORDER.filter(p => found.includes(p)), ...found.filter(p => !ORDER.includes(p))];
  const drift = [];
  const units = [];
  for (const p of perspectives) {
    units.push(loadUnit(p, null, join(base, p), drift, rel));
    for (const f of subdirs(join(base, p))) units.push(loadUnit(p, f, join(base, p, f), drift, rel));
  }
  const globals = new Map(units.filter(u => !u.feature).map(u => [u.perspective, u]));
  linkCites(units, globals);
  parkSpecGaps(join(base, 'TODO.md'), units, drift);
  return { perspectives, units, drift, rel };
}

function loadUnit(perspective, feature, dir, drift, rel) {
  const unit = { perspective, feature, id: feature ? `${perspective}/${feature}` : perspective, dir, rules: [] };
  for (const layer of LAYERS) {
    const specFile = join(dir, feature ? `${layer}.md` : 'AGENTS.md');
    const trackFile = join(dir, `${layer}-tracking.md`);
    const spec = existsSync(specFile) ? specRules(specFile, feature ? layer : null).filter(r => r.layer === layer) : [];
    const entries = new Map();
    for (const e of existsSync(trackFile) ? trackingEntries(trackFile) : []) {
      const key = e.title.toLowerCase();
      if (entries.has(key)) drift.push(`${rel(trackFile)}: "${e.title}" has two entries`);
      else entries.set(key, e);
    }
    for (const r of spec) {
      const key = r.title.toLowerCase();
      const entry = entries.get(key) ?? null;
      entries.delete(key);
      if (!entry) drift.push(`${rel(specFile)}: "${r.title}" has no tracking entry`);
      else if (entry.removed) drift.push(`${rel(trackFile)}: "${r.title}" is flagged Removed: but the rule is still in the spec`);
      unit.rules.push({ ...r, unit, specFile, entry });
    }
    for (const entry of entries.values()) {
      if (entry.removed) unit.rules.push({ title: entry.title, layer, body: '', unit, specFile: null, entry });
      else drift.push(`${rel(trackFile)}: "${entry.title}" has no rule and no Removed:`);
    }
  }
  return unit;
}

const pending = r => Boolean(r.entry && (!r.entry.done || r.entry.removed));

function state(r) {
  if (r.entry.removed) return 'removed';
  if (r.entry.changed) return 'changed';
  return r.entry.paths.length ? 'interrupted' : 'new';
}

// Anchor slug per **Anchors** of perspectives/AGENTS.md: accents stay.
const slug = title => title.toLowerCase().replace(/[^\p{L}\p{N}_ -]/gu, '').replace(/ /g, '-');

// Direct cites of every rule — links, bold titles, and Dependency: — kept when the target is `[ ]`.
function linkCites(units, globals) {
  const byKey = new Map();
  const bySlug = new Map();
  for (const r of units.flatMap(u => u.rules).filter(r => r.specFile)) {
    byKey.set(r.title.toLowerCase(), [...(byKey.get(r.title.toLowerCase()) ?? []), r]);
    if (!bySlug.has(r.specFile)) bySlug.set(r.specFile, new Map());
    bySlug.get(r.specFile).set(slug(r.title), r);
  }
  const byTitle = (key, from) => {
    const candidates = byKey.get(key) ?? [];
    return candidates.find(c => c.specFile === from.specFile)
      ?? candidates.find(c => c.unit === from.unit)
      ?? candidates.find(c => c.unit === globals.get(from.unit.perspective))
      ?? candidates.find(c => c.unit === globals.get('development'))
      ?? (candidates.length === 1 ? candidates[0] : candidates.length ? 'ambiguous' : null);
  };
  for (const r of units.flatMap(u => u.rules)) {
    const targets = new Set();
    const ambiguous = new Set();
    const add = (hit, text) => (hit === 'ambiguous' ? ambiguous.add(text) : hit && targets.add(hit));
    const dependency = r.entry?.dependency ?? [];
    const text = [r.body, ...dependency].join('\n');
    for (const [, href] of r.specFile ? text.matchAll(/\]\(([^)\s]+)\)/g) : []) {
      const [file, anchor] = href.split('#');
      if (!anchor) continue;
      const target = file ? resolve(dirname(r.specFile), safeDecode(file)) : r.specFile;
      add(bySlug.get(target)?.get(safeDecode(anchor).toLowerCase()));
    }
    for (const [, title] of r.specFile ? text.matchAll(/\*\*([^*]+)\*\*/g) : []) add(byTitle(title.trim().toLowerCase(), r), title.trim());
    for (const line of r.specFile ? dependency : []) {
      const plain = line.replace(/^Dependency:\s*/, '');
      if (/\*\*|\]\(/.test(plain)) continue;
      const title = plain.replace(/\s*\(.*$/, '').replace(/[.\s]+$/, '');
      add(byTitle(title.toLowerCase(), r), title);
    }
    targets.delete(r);
    r.cites = [...targets].filter(pending);
    r.ambiguous = [...ambiguous];
  }
}

function safeDecode(text) {
  try {
    return decodeURIComponent(text);
  } catch {
    return text;
  }
}

// `## Spec gap` items of TODO.md park the `[ ]` rule they name.
function parkSpecGaps(file, units, drift) {
  if (!existsSync(file)) return;
  const section = sections(readFileSync(file, 'utf8')).find(s => s.level === 2 && s.title.toLowerCase() === 'spec gap');
  for (const line of section?.lines ?? []) {
    if (!/^\s*-\s/.test(line)) continue;
    const m = line.match(/^\s*-\s*\[[ xX]\]\s*\*\*(.+?)\*\*\s*\(`([^`]+)`(?:,\s*(policy|mechanism))?\)\s*(.*)$/);
    if (!m) {
      drift.push(`perspectives/TODO.md: spec gap out of shape: ${line.trim()}`);
      continue;
    }
    const [, title, owner, layer, reason] = m;
    const rule = units
      .find(u => u.id === owner)
      ?.rules.find(r => pending(r) && r.title.toLowerCase() === title.toLowerCase() && (!layer || r.layer === layer));
    if (rule) rule.parked = reason.trim() || 'spec gap';
    else drift.push(`perspectives/TODO.md: spec gap "${title}" (${owner}) matches no [ ] rule`);
  }
}

const unitOrder = (a, b) => (a.feature ? 1 : 0) - (b.feature ? 1 : 0) || (a.feature ?? '').localeCompare(b.feature ?? '');

// Units of one perspective grouped by cycle (Tarjan), groups in dependency order, ties by unitOrder.
function topoGroups(units) {
  const inside = new Set(units);
  const deps = new Map(units.map(u => [u, new Set(u.rules.filter(pending).flatMap(r => r.cites.map(c => c.unit)).filter(v => v !== u && inside.has(v)))]));
  const index = new Map();
  const low = new Map();
  const stack = [];
  const groups = [];
  const visit = u => {
    index.set(u, index.size);
    low.set(u, index.get(u));
    stack.push(u);
    for (const v of deps.get(u)) {
      if (!index.has(v)) {
        visit(v);
        low.set(u, Math.min(low.get(u), low.get(v)));
      } else if (stack.includes(v)) low.set(u, Math.min(low.get(u), index.get(v)));
    }
    if (low.get(u) === index.get(u)) groups.push(stack.splice(stack.indexOf(u)).sort(unitOrder));
  };
  for (const u of units) if (!index.has(u)) visit(u);
  const groupOf = new Map(groups.flatMap((g, i) => g.map(u => [u, i])));
  const needs = groups.map((g, i) => new Set(g.flatMap(u => [...deps.get(u)].map(v => groupOf.get(v))).filter(j => j !== i)));
  const order = [];
  const done = new Set();
  while (order.length < groups.length) {
    const next = groups
      .map((_, i) => i)
      .filter(i => !done.has(i) && [...needs[i]].every(j => done.has(j)))
      .sort((a, b) => unitOrder(groups[a][0], groups[b][0]))[0];
    done.add(next);
    order.push(groups[next]);
  }
  return order;
}

// Rules of `units` that can run: `[ ]`, not parked, every pending cite runs inside `units`.
function settle(units) {
  const inside = new Set(units);
  const run = new Set(units.flatMap(u => u.rules.filter(r => pending(r) && !r.parked)));
  for (let changed = true; changed; ) {
    changed = false;
    for (const r of run) {
      if (r.cites.some(c => !inside.has(c.unit) || !run.has(c))) {
        run.delete(r);
        changed = true;
      }
    }
  }
  return run;
}

function nextBatch(perspectives, units) {
  for (const p of perspectives) {
    const own = units.filter(u => u.perspective === p && u.rules.some(pending));
    const batch = [];
    const seen = [];
    for (const group of topoGroups(own)) {
      if (batch.length >= CAP) break;
      const run = settle([...seen, ...group]);
      const kept = group.filter(u => u.rules.some(r => run.has(r)));
      if (batch.length && batch.length + kept.length > CAP) break;
      seen.push(...group);
      batch.push(...kept);
    }
    if (batch.length) return { perspective: p, units: batch, inside: new Set(seen), run: settle(seen) };
  }
  return null;
}

function why(r, inside, run) {
  if (r.parked) return `parked: ${r.parked}`;
  const c = r.cites.find(c => !inside.has(c.unit) || !run.has(c));
  if (!c) return 'waits';
  const what = c.parked
    ? 'is parked'
    : inside.has(c.unit)
      ? 'waits'
      : c.unit.perspective === r.unit.perspective
        ? 'is outside the batch'
        : `is in ${c.unit.perspective}`;
  return `cites ${c.title} (${c.unit.id}), which ${what}`;
}

// Removed: first, then policy before mechanism, then file order.
const ordered = u => u.rules.filter(pending).sort((a, b) => Boolean(b.entry.removed) - Boolean(a.entry.removed) || LAYERS.indexOf(a.layer) - LAYERS.indexOf(b.layer));

function describe(r) {
  const n = r.entry.paths.length;
  const bits = [r.layer, state(r)];
  if (n) bits.push(`${n} path${n > 1 ? 's' : ''}`);
  return bits.join(', ');
}

function render({ perspectives, units, drift, rel }, opts) {
  const out = [];
  const open = units.filter(u => u.rules.some(pending));
  const label = u => (u.feature ? `${u.id} — ${rel(u.dir)}` : `${u.id} (global rules) — ${rel(join(u.dir, 'AGENTS.md'))}`);
  if (!open.length) out.push('No pending entries.');
  if (open.length && !opts.batch) {
    out.push('# Pending');
    for (const p of perspectives.filter(p => open.some(u => u.perspective === p))) {
      out.push('', `## ${p}`, '');
      for (const u of open.filter(u => u.perspective === p)) {
        out.push(`- ${label(u)}`);
        for (const r of ordered(u)) {
          const bits = [describe(r)];
          if (r.cites.length) bits.push(`cites ${r.cites.map(c => `${c.title} (${c.unit.id})`).join('; ')}`);
          if (r.ambiguous.length) bits.push(`ambiguous cite ${r.ambiguous.map(t => `"${t}"`).join(', ')}`);
          if (r.parked) bits.push(`parked: ${r.parked}`);
          out.push(`  - [ ] ${r.title} — ${bits.join(' — ')}`);
        }
      }
    }
    out.push('');
  }
  if (open.length) {
    const batch = nextBatch(perspectives, units);
    if (batch) {
      out.push(`# Next batch: ${batch.perspective}, ${batch.units.length} unit${batch.units.length > 1 ? 's' : ''}`, '');
      batch.units.forEach((u, i) => {
        out.push(`${i + 1}. ${label(u)}`);
        for (const r of ordered(u)) {
          if (batch.run.has(r)) out.push(`   run  ${r.title} — ${describe(r)}`);
          else out.push(`   wait ${r.title} — ${r.layer} — ${why(r, batch.inside, batch.run)}`);
        }
      });
    } else {
      out.push('# Next batch: none', '', 'Every [ ] rule waits:');
      for (const p of perspectives) {
        const own = open.filter(u => u.perspective === p);
        const inside = new Set(own);
        const run = settle(own);
        for (const u of own) for (const r of ordered(u)) out.push(`- ${u.id}: ${r.title} — ${why(r, inside, run)}`);
      }
    }
  }
  if (drift.length) out.push(...(out.at(-1) === '' ? [] : ['']), '# Drift', '', ...drift.map(d => `- ${d}`));
  return out.join('\n').replace(/\n+$/, '');
}

const opts = parseArgs(process.argv.slice(2));
process.stdout.write(render(loadTree(opts.root), opts) + '\n');
