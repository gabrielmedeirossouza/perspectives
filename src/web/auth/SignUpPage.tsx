import { Link, useNavigate } from "react-router";
import { signUp } from "./api";
import { CredentialsForm } from "./CredentialsForm";
import { validateSignUp } from "./validate";

export const EMAIL_TAKEN = "This email already has an account";

export function SignUpPage() {
  const navigate = useNavigate();
  async function submit(email: string, password: string) {
    const result = await signUp(email, password);
    if (result.ok) {
      navigate("/");
      return undefined;
    }
    return result.error === "email_taken" ? EMAIL_TAKEN : "Something went wrong";
  }
  return (
    <>
      <CredentialsForm title="Create your account" submitLabel="Create account" validate={validateSignUp} submit={submit} />
      <p>
        Already a member? <Link to="/sign-in">Sign in</Link>
      </p>
    </>
  );
}
