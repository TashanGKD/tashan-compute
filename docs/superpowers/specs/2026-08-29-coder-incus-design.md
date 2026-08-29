# Tashan Compute — Coder + Incus Design

Date: 2026-08-29

Status: approved architecture replacing the custom object-storage and executor platform before that implementation began.

## Product boundary

Tashan Compute is a public Codex Skill and thin `tcompute` CLI that installs and operates a self-hosted Coder deployment. Coder owns user/password authentication, sessions, workspace lifecycle, SSH, terminal access, workspace sharing and application tunnels. Incus owns the isolated Linux system containers or virtual machines in which users receive a complete shell.

The previously implemented custom Foundation remains undeployed reference code. Production must not run both authentication systems or ask users to choose between them.

## User model

- A platform administrator creates password users in Coder. Users cannot self-register.
- Users authenticate with `coder login https://compute.tashan.chat`; `tcompute login` delegates to that command while requiring operating-system keyring storage.
- A personal space is a persistent Coder workspace owned by one user.
- An organization space is a persistent Coder workspace shared with explicitly named users. OSS workspace sharing supplies `use` and `admin` access; read-only file access is enforced inside the workspace only if later required.
- Removing a shared user must be followed by a workspace restart because Coder documents that access removal takes effect after restart.

## Isolation model

Each workspace is an unprivileged Incus system container with an isolated UID/GID map. Root inside the container maps to an unprivileged host ID. Users never receive the Incus socket, Incus client certificate, AUP host SSH, Docker socket or Coder template-edit permission.

The Coder provisioner runs as a dedicated Linux service account without sudo. It receives only confined Incus project access. Templates are maintained in Git and pushed by platform administrators; users cannot submit Terraform.

Initial resource profiles reserve platform and emergency capacity. Each workspace has hard CPU, memory, process and disk settings. Containers can access the public Internet, while host, Incus daemon, Docker API, cloud metadata, private management networks and other workspace networks are blocked.

## Shell and computation

`tcompute shell <workspace>` delegates to `coder ssh <workspace>`. The user receives a normal interactive Linux shell and can install packages, compile code, run Python/Node/Go/Rust/C/C++, start databases and long-running daemons inside the workspace.

Docker/OCI builds run inside the Incus container only after `security.nesting=true` and a rootless container engine or BuildKit path passes escape and resource tests. Users cannot use the AUP host Docker daemon.

## Files and collaboration

Workspace files persist on Incus storage volumes. Coder SSH configuration enables standard `ssh`, `scp` and `rsync`; `tcompute sync` is a thin safe wrapper whose default is dry-run. Organization collaboration occurs by multiple authenticated Coder users entering the same shared workspace, not by mounting a custom MinIO filesystem.

## Services

Authenticated workspace applications and port forwarding use Coder. Workspace apps use a separate wildcard domain to avoid same-origin risk. Anonymous publication remains an explicit `tcompute service public` operation implemented by a small platform-controlled reverse proxy policy, never by exposing the container or host port directly.

## Open-source boundary

The implementation depends only on Coder OSS capabilities required here: password users, personal workspaces, CLI login, SSH, workspace apps and workspace sharing. Coder multi-Organization and custom-role features are Premium and are not runtime dependencies. “Organization” in Tashan Compute is represented by a shared workspace and an access list.

## AUP baseline

The verified AUP host is Ubuntu 24.04 with kernel 6.14, 32 logical CPUs, about 62 GiB memory, roughly 1 TiB free disk and `/dev/kvm`. Incus, Coder, `newuidmap` and `newgidmap` are not installed; passwordless non-interactive sudo is available to the current operator. Ubuntu provides Incus 6.0 LTS and uidmap packages.

## Acceptance

1. A new user with no Tailscale installs the public Skill and CLI, then logs into Coder with an administrator-created temporary password and changes it.
2. Alice and Bob each receive an isolated personal workspace and cannot access one another’s workspace.
3. Both can SSH into a shared organization workspace and edit/run the same project.
4. Python, Node.js, a compiled program, a container build, a database and a daemon run inside the isolated workspace.
5. Resource exhaustion remains within workspace limits; host/other workspace/metadata/daemon access is rejected.
6. A private workspace app requires login; explicit publication permits anonymous HTTPS and can be reversed.
7. Source-only access to the public repository grants neither a Coder account nor Incus/AUP administration.
