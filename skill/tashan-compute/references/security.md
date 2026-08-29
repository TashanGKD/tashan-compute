# Safety boundaries

- A workspace root user maps to a non-root host UID. Host/private/cloud-metadata networks and host Docker/Incus sockets are blocked; public Internet egress is allowed.
- Use `/home/coder` for persistent files. Paths outside it are ephemeral container state and disappear when compute is recreated.
- `workspace delete` is permanent and requires the exact target plus `--yes`.
- `sync` never enables `--delete`; it dry-runs unless `--apply` is explicit.
- Removing a shared member automatically restarts the workspace so revocation takes effect immediately.
- `service private` requires owner login, `service authenticated` requires a platform login, and `service public` permits anonymous Internet access.
- Never substitute AUP SSH, Tailscale, an Incus socket, a host path or a Docker socket for a missing product command.
