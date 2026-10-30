import { Link, useNavigate } from "react-router";
import { signIn } from "./api";
import { CredentialsForm } from "./CredentialsForm";
import { validateSignIn } from "./validate";

export const INVALID_CREDENTIALS = "Invalid email or password";

export function SignInPage() {
  const navigate = useNavigate();
  async function submit(email: string, password: string) {
    const result = await signIn(email, password);
    if (result.ok) {
      navigate("/");
      return undefined;
    }
    return result.status === 401 ? INVALID_CREDENTIALS : "Something went wrong";
  }
  return (
    <>
      <CredentialsForm title="Sign in" submitLabel="Sign in" validate={validateSignIn} submit={submit} />
      <p>
        New here? <Link to="/sign-up">Create an account</Link>
      </p>
    </>
  );
}
