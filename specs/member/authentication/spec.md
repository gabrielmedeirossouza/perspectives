# Authentication

The sign-up and sign-in pages, the sign-out action, and where a member without a session lands.

## Pages

### Sign-up page

The sign-up page, titled "Create your account", **must** show an email field, a password field, a "Create account" button, and a "Sign in" link to the sign-in page. A submit with an empty field, an email without `@`, or a password shorter than **Password length** allows **must** show the error under that field and **must not** be sent. A sign-up refused because the email already has an account **must** show "This email already has an account" above the form. A successful sign-up **must** open the home page, signed in.

### Sign-in page

The sign-in page, titled "Sign in", **must** show an email field, a password field, a "Sign in" button, and a "Create an account" link to the sign-up page. A submit with an empty field **must** show the error under that field and **must not** be sent. Refused credentials **must** show "Invalid email or password" above the form, whichever of the two was wrong. A successful sign-in **must** open the home page.

### Unexpected failure message

A sign-up or sign-in that fails for a reason no rule names **must** show "Something went wrong" above the form.

## Session

### Sign-out action

Every page of a signed-in member **must** show, in the header, the member's email and "Sign out". Activating "Sign out" **must** end the session and open the sign-in page.

### Signed-out landing

A member page opened without a session — none, ended, or expired — **must** open the sign-in page instead.
