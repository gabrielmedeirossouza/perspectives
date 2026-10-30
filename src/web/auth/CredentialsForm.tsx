import { useState, type FormEvent } from "react";
import type { FieldErrors } from "./validate";

export interface CredentialsFormProps {
  title: string;
  submitLabel: string;
  validate: (email: string, password: string) => FieldErrors;
  /** Sends the form; returns the message to show above it, or nothing when done. */
  submit: (email: string, password: string) => Promise<string | undefined>;
}

/** The email and password form shared by the sign-up and sign-in pages. */
export function CredentialsForm({ title, submitLabel, validate, submit }: CredentialsFormProps) {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [errors, setErrors] = useState<FieldErrors>({});
  const [message, setMessage] = useState<string | undefined>();

  async function onSubmit(event: FormEvent) {
    event.preventDefault();
    const found = validate(email, password);
    setErrors(found);
    setMessage(undefined);
    if (found.email || found.password) return;
    setMessage(await submit(email, password));
  }

  return (
    <main>
      <h1>{title}</h1>
      <form onSubmit={onSubmit} noValidate>
        {message && <p role="alert">{message}</p>}
        <label>
          Email
          <input name="email" type="email" value={email} onChange={(e) => setEmail(e.target.value)} />
        </label>
        {errors.email && <p>{errors.email}</p>}
        <label>
          Password
          <input name="password" type="password" value={password} onChange={(e) => setPassword(e.target.value)} />
        </label>
        {errors.password && <p>{errors.password}</p>}
        <button type="submit">{submitLabel}</button>
      </form>
    </main>
  );
}
