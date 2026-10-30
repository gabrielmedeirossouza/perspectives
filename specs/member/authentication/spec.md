# Authentication

The sign-up and sign-in pages, the sign-out action, and where a member without a session lands.

## Pages

### Sign-up page

The sign-up page **must** show an email field, a password field, and a "Create account" button. A submit with an empty field, or with a password shorter than **Minimum password length** allows, **must** show the error under that field and **must not** be sent. A sign-up refused because the email already has an account **must** show "This email already has an account" above the form. A successful sign-up **must** open the home page, signed in.

### Sign-in page

The sign-in page **must** show an email field, a password field, and a "Sign in" button. A submit with an empty field **must** show the error under that field and **must not** be sent. Refused credentials **must** show "Invalid email or password" above the form, whichever of the two was wrong. A successful sign-in **must** open the home page.

## Session

### Sign-out action

Every page of a signed-in member **must** show "Sign out" in the header. Activating it **must** end the session and open the sign-in page.

### Signed-out landing

A member page opened without a session — none, ended, or expired — **must** open the sign-in page instead.
