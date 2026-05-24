# Security Policy

## Supported Versions

We currently support the following versions with security updates:

| Version | Supported          |
| ------- | ------------------ |
| 0.5.x   | :white_check_mark: |
| < 0.5   | :x:                |

## Reporting a Vulnerability

We take the security of Crescent Moon Visibility seriously.

If you discover a security vulnerability, please report it responsibly:

1. **Do not** create a public GitHub issue.
2. Email the maintainers at: security@jim-fun.com (or open a private security advisory on GitHub if available).
3. Include as much detail as possible (steps to reproduce, affected versions, potential impact).

We will acknowledge receipt of your report within 48 hours and aim to provide a fix or mitigation timeline within 7 days for confirmed vulnerabilities.

## Security Best Practices for Users

- Always verify downloaded binaries using the provided `checksums.txt` and Cosign signatures.
- Build from source when possible, especially on macOS or non-Linux platforms.
- Keep your OpenCL drivers and system libraries up to date.

Thank you for helping keep the project secure!