export const MIN_PASSWORD_LENGTH = 8;

export interface FieldErrors {
  email?: string;
  password?: string;
}

/** Errors shown under the fields of the sign-in page; an empty object means the form may be sent. */
export function validateSignIn(email: string, password: string): FieldErrors {
  const errors: FieldErrors = {};
  if (email.trim() === "") errors.email = "Enter your email";
  if (password === "") errors.password = "Enter your password";
  return errors;
}

/** Errors shown under the fields of the sign-up page: empty fields and a short password. */
export function validateSignUp(email: string, password: string): FieldErrors {
  const errors = validateSignIn(email, password);
  if (!errors.password && password.length < MIN_PASSWORD_LENGTH) errors.password = `Use at least ${MIN_PASSWORD_LENGTH} characters`;
  return errors;
}
