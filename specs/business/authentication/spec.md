# Authentication

Who may hold an account, how a member proves an account is theirs, and how long a session lasts.

**Out of scope:** verifying that a member owns the email; recovering a forgotten password.

## Account

### One account per email

An email address **must** identify at most one account. A sign-up with an email that already identifies an account **must** be refused.

### Email shape

A sign-up email **must** contain `@` and have at most 254 characters; a sign-up with any other email **must** be refused.

### Password length

A password **must** have at least 8 and at most 128 characters. No other rule constrains it.

### Credentials prove the account

A member **must** sign in with the email and the password of the account. A sign-in with a wrong email or a wrong password **must** be refused, and the refusal **must not** tell which of the two was wrong.

## Session

### Session lifetime

A session **must** start at sign-up and at sign-in, **must** end at sign-out, and **must** end after 7 days without a request from the member. A request on an ended session **must** be treated as a request without a session.

### Data isolation by account

A request with a session **must** reach only the data of the account that session belongs to.
