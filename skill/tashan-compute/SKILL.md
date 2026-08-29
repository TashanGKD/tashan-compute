---
name: tashan-compute
description: Use when installing or operating Tashan Compute personal or shared organization workspaces, remote Linux shells, file sync, code and container builds, resident services, databases, daemons, or administrator-managed accounts.
---

# Tashan Compute

Use `tcompute` for every user operation. Ordinary users connect through public HTTPS and Coder workspace tunnels; never ask them for AUP SSH, Tailscale, Incus or host Docker access.

## Start

1. If `tcompute` is missing, run `bash scripts/install-cli.sh --check`. Install only after the user requests it: `bash scripts/install-cli.sh --install`.
2. Run `tcompute <command> --help` before constructing unfamiliar flags.
3. Inspect supported operations with `tcompute capability list`.

Accounts are administrator-created. Use `tcompute login --email <email>` and its hidden password prompt. Never request a password or token in chat. Read [authentication.md](references/authentication.md) for first-login and reset handling.

## Spaces and compute

```text
tcompute personal create <name>
tcompute workspace list
tcompute org create <name> --admin <existing-user>  # platform admin only
tcompute org list
tcompute org member add <workspace> <username>
tcompute shell <workspace>
tcompute shell <owner>/<shared-workspace>
```

The shell is root inside an unprivileged Incus container, not on the AUP host. Python, Node.js, Go, Rust, C/C++, PostgreSQL, Redis, Podman and the Docker-compatible `docker build` command are installed. Persistent files belong under `/home/coder`; personal volumes are 50 GiB and platform-created organization volumes are 500 GiB.

## Files

`sync push` and `sync pull` are dry-run by default. Add `--apply` only after checking the named workspace, local path and remote path. Remote paths are always relative to `/home/coder`.

## Resident service

Put an executable launcher at `/home/coder/.tcompute/service`; it is restarted whenever the workspace is recreated. Port 8000 receives an HTTPS workspace hostname.

- Default: `tcompute service private <workspace>` — owner login required.
- Organization login: `tcompute service authenticated <workspace>`.
- Anonymous Internet: `tcompute service public <workspace>`.

`service public` is an explicit security change. State clearly that anyone on the Internet can access the service, obtain approval for the exact workspace, and show the reversal command `service private`.

Read [security.md](references/security.md) before deletion, publication or administrator operations.
