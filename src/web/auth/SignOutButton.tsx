import { useNavigate } from "react-router";
import { signOut } from "./api";

/** Ends the session and opens the sign-in page. */
export function SignOutButton() {
  const navigate = useNavigate();
  async function onClick() {
    await signOut();
    navigate("/sign-in");
  }
  return (
    <button type="button" onClick={onClick}>
      Sign out
    </button>
  );
}
