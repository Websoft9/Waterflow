# Security Policy

## Supported Versions

We release patches for security vulnerabilities for the following versions:

| Version | Supported          |
| ------- | ------------------ |
| 1.x.x   | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

We take the security of Waterflow seriously. If you believe you have found a security vulnerability, please report it to us as described below.

### Please do NOT:

- Open a public GitHub issue
- Discuss the vulnerability in public forums, chat rooms, or social media

### Please DO:

1. **Email us directly** at security@websoft9.com with:
   - Type of vulnerability
   - Full paths of source file(s) related to the vulnerability
   - Location of the affected source code (tag/branch/commit or direct URL)
   - Any special configuration required to reproduce the issue
   - Step-by-step instructions to reproduce the issue
   - Proof-of-concept or exploit code (if possible)
   - Impact of the issue, including how an attacker might exploit it

2. **Allow us time to respond** - We will acknowledge your email within 48 hours and aim to send a more detailed response within 7 days.

3. **Work with us** - We may ask for additional information or guidance.

## What to Expect

- **Acknowledgment**: Within 48 hours
- **Initial Assessment**: Within 7 days
- **Fix & Disclosure Timeline**: We will work with you to understand the timeline
- **Credit**: We will publicly credit you for responsibly disclosing the issue (unless you prefer to remain anonymous)

## Security Update Process

1. The security team will investigate and validate the report
2. If confirmed, we will:
   - Develop a fix
   - Prepare a security advisory
   - Release a patched version
   - Publish the security advisory

## Security Best Practices

When using Waterflow:

- Always use the latest stable version
- Review and validate YAML configurations before execution
- Run with least privilege principle
- Keep dependencies up to date
- Enable security scanning in your CI/CD pipeline
- Monitor security advisories: https://github.com/Websoft9/Waterflow/security/advisories

## GPG Key

For sensitive reports, you may encrypt your email using our GPG key:

```
(GPG key to be added)
```

## Bug Bounty Program

We currently do not have a bug bounty program. However, we deeply appreciate security researchers who help us keep our users safe.

Thank you for helping keep Waterflow and our users safe!
