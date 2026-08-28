---
name: tashan-compute
description: Use when a user wants to install or operate the Tashan Compute CLI for authorized personal or organization files, code execution, builds, services, databases, daemons, devices, or administrator-created accounts.
---

# Tashan Compute

Use the public `tcompute` CLI as the sole product interface. Installing this Skill or CLI does not create an account, grant an organization membership, expose AUP SSH, or provide administrator access.

## Before a command

1. If `tcompute` is unavailable, run `scripts/install-cli.sh --check`. After the user asks to install, run `scripts/install-cli.sh --install`.
2. Read `tcompute <group> --help` for current flags. Do not invent commands from this file.
3. Use `tcompute capability list --json` when deciding whether the connected server supports an operation.

## Authentication

Accounts are created by a platform administrator. For login, password change, reset consequences and device sessions, read [references/authentication.md](references/authentication.md).

Never ask the user to paste a password, Token or Secret into chat. Use the CLI hidden prompt. Use `--password-stdin` only when the calling environment supplies protected stdin and will not log it.

## Safety

Read [references/security.md](references/security.md) before deletion, overwrite, public service access, Secret changes or administrator operations.

- Keep explicit `--space` and `--org` selectors; never infer a target from conversation history.
- Preserve dry-run and confirmation gates. Do not add `--yes` unless the user approved the exact action and target.
- Treat stdout JSON as data and stderr as diagnostics. Never reproduce stored credentials in the response.
- Do not use SSH, Tailscale, Docker commands or host paths as a substitute for a missing platform capability.
