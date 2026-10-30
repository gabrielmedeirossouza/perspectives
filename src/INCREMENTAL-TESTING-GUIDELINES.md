# Incremental testing guidelines

Stateful collection of reusable unit-testing practices learned from bug
autopsies in `src/`.

## Guidelines

<!-- Add each new reusable testing practice here after its autopsy. -->

- A response contract that holds for "every response" **must** be tested on the responses the framework answers by itself: an unknown route, a body that cannot be parsed, and an unhandled exception, besides the routes' own replies.
