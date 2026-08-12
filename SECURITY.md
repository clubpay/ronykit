# Security Policy

## Supported versions

RonyKit is a Go workspace of independently tagged modules (`kit`, `rony`, `intent`, `stub`, `ronyup`, and the `std/*` and `x/*` adapters). Only the **latest tagged release of each module** is supported; fixes are published as a new tag rather than backported to older ones.

## Reporting a vulnerability

**Do not open a public issue for security reports.**

Use GitHub's private vulnerability reporting:

https://github.com/clubpay/ronykit/security/advisories/new

If that form is not available, open a regular issue that contains **no exploit details** — just a request for a private channel — and a maintainer will open a private advisory to continue there.

Please include, where you can:

- the affected module and version/tag,
- a description of the impact,
- steps or a minimal program that reproduces the issue.

## Disclosure

We coordinate disclosure with the reporter. Once a fix is tagged, the advisory is published and the reporter is credited unless they ask otherwise.
