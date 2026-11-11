# Security Policy

## Supported Versions

Only the latest release of gostrata receives security fixes. Please upgrade
before reporting an issue to confirm it still reproduces.

## Reporting a Vulnerability

Please **do not open a public issue** for security problems.

Report privately in one of these ways:

1. **GitHub private vulnerability reporting** (preferred):
   [Security → Report a vulnerability](https://github.com/skryfon-org/gostrata/security/advisories/new)
2. **Email**: security@skryfon.com

Include, where possible:

- A description of the issue and its impact
- Steps to reproduce, or a proof of concept
- The affected version and your OS

## What to Expect

- Acknowledgement within 7 days
- An assessment and plan within 14 days
- Credit in the release notes once a fix ships, unless you prefer to stay anonymous

## Scope

In scope: the gostrata CLI, the code it generates for new projects (including
the Dockerfile), and the install scripts (`install.sh`, `install.ps1`).

Out of scope: vulnerabilities in third-party dependencies with no impact on
gostrata (report those upstream), and issues in projects you generated after you
modified them.
