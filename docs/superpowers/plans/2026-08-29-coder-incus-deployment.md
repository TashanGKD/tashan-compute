# Coder + Incus Deployment Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans. Core implementation stays inline; subagents may only audit and run fresh-user tests.

**Goal:** Deploy a public, no-Tailscale Coder service on AUP whose workspaces are unprivileged Incus Linux environments, then ship a thin Skill/CLI that creates, shares, enters and synchronizes those environments.

**Architecture:** Coder OSS is the only production identity and workspace control plane. A dedicated non-sudo service account runs Coder and a confined Incus provisioner; Terraform templates create persistent unprivileged Incus containers. ECS terminates HTTPS and forwards one persistent tunnel to Coder; workspace apps use a separate wildcard hostname.

**Tech Stack:** Coder OSS v2.35.6, Incus 6.0 LTS, terraform-provider-incus v1.2.0, PostgreSQL 17, OpenSSH/rsync, Go wrapper CLI, ECS Nginx/TLS.

---

### Task 1: Pin upstream artifacts and safe deployment scripts

**Files:** `deploy/versions.env`, `deploy/install-aup-prerequisites.sh`, `deploy/install-coder.sh`, matching `*.self-test.sh`.

- [ ] Write tests proving no-argument mode is help-only, wrong SHA-256 fails, unexpected package/version fails, and production mutation requires `--apply --host aup-test-01`.
- [ ] Run tests and observe missing scripts.
- [ ] Pin Coder `2.35.6`, Incus Ubuntu package `6.0.0-1ubuntu0.3`, uidmap `1:4.13+dfsg1-4ubuntu3`, and Incus provider `1.2.0`. Fetch checksums from the official releases rather than embedding secrets.
- [ ] Implement dry-run defaults and explicit host/commit evidence. Apt operations install only `incus`, `incus-client`, `uidmap`, `qemu-system-x86` and required networking helpers.
- [ ] Verify scripts in a local fake-command harness, commit `chore(deploy): pin coder and incus prerequisites`.

### Task 2: Initialize confined Incus runtime

**Files:** `deploy/incus-preseed.yaml`, `deploy/configure-incus.sh`, `deploy/verify-incus-isolation.sh`, self-tests.

- [ ] Encode negative tests: privileged container, missing isolated idmap, Incus socket in workspace, host bind mount, unlimited CPU/memory/processes/disk, host/private/metadata connectivity.
- [ ] Install packages on AUP after a read-only preflight and initialize a dedicated `tcompute` storage pool/network/profile. Prefer an Incus-supported quota-capable pool; fail closed if disk quota is not enforced.
- [ ] Create dedicated service user without sudo and grant only confined Incus access. Never add it to `incus-admin` unless a test proves no safer provisioning path exists and the user explicitly approves that escalation.
- [ ] Launch a disposable workspace, confirm container root maps to a non-root host UID, resource limits apply, public egress works and forbidden targets fail.
- [ ] Destroy only the named disposable instance and record evidence; commit `security(runtime): verify incus workspace isolation`.

### Task 3: Deploy Coder OSS control plane

**Files:** `deploy/compose.coder.yml`, `deploy/coder.env.example`, `deploy/tcompute-coder.service`, `deploy/deploy-coder.sh`, tests.

- [ ] Test that source defaults bind loopback, no password/token appears in compose/systemd/logs, and no Docker/Incus socket is exposed to users.
- [ ] Deploy dedicated PostgreSQL and Coder v2.35.6 under `/home/aup/tashan-compute`; secrets are generated on AUP into `0600/0640` files outside Git.
- [ ] Run Coder as the dedicated non-sudo service account. Configure access URL, wildcard app URL, password authentication and session limits; disable user template administration.
- [ ] Verify local health and restart recovery. Commit `feat(coder): deploy isolated control plane`.

### Task 4: Create Incus Coder template

**Files:** `templates/incus-workspace/main.tf`, `variables.tf`, `cloud-init.yaml.tftpl`, `README.md`, template tests.

- [ ] Write static and live tests rejecting privileged/nesting-by-default profiles, host paths, missing limits and absent Coder agent.
- [ ] Use `lxc/incus` provider 1.2.0 and Coder provider. Create an unprivileged Ubuntu 24.04 container, persistent home volume, Coder agent, metadata labels and capped resources.
- [ ] Provide parameters for personal/shared kind, CPU, memory and disk within administrator-defined bounds. `security.nesting` is false by default.
- [ ] Push template as platform admin, create a disposable workspace, connect with `coder ssh`, restart it and verify persistence.
- [ ] Commit `feat(template): add incus coder workspace`.

### Task 5: Replace custom CLI runtime with thin Coder wrapper

**Files:** `internal/codercli`, `internal/cli`, `cmd/tcompute`, capability/Skill bindings and tests.

- [ ] Write failing command tests for `login`, `shell`, `personal create`, `org create`, `org member add/remove`, `sync`, `service private/public`, including password/token output rejection.
- [ ] Execute Coder as an argv array with explicit environment and stdout/stderr separation. Do not parse prose when `--output json` exists.
- [ ] Require Coder keyring storage; on Linux fail with guidance if no keyring instead of accepting Coder’s default plaintext session file.
- [ ] Map personal workspace ownership and shared workspace access; member removal automatically restarts the workspace before reporting success.
- [ ] Make `sync` dry-run by default and invoke rsync only through generated Coder SSH configuration.
- [ ] Update capability/route gates to reflect delegated Coder operations; commit `feat(cli): delegate workspaces to coder`.

### Task 6: Full shell, build and long-running workload validation

**Files:** `images/workspace/Containerfile`, `scripts/verify-workspace-tools.sh`, tests.

- [ ] Build a pinned workspace image with Python, Node.js, Go, Rust, C/C++, Git, rsync, process supervisor and database clients.
- [ ] Validate package installation as container root, persistent home, Python/Node/compiled runs, PostgreSQL/Redis in-workspace processes and daemon restart.
- [ ] Enable nested rootless BuildKit/Podman only in a separate tested template/profile. Reject host Docker socket and privileged nesting.
- [ ] Record CPU/memory/pids/disk enforcement and host/private network rejections. Commit `feat(runtime): validate complete workspace shell`.

### Task 7: Public HTTPS and workspace applications

**Files:** `deploy/nginx/compute.tashan.chat.conf`, `deploy/start-coder-tunnel.sh`, `deploy/verify-public-access.sh`, tests.

- [ ] Reserve independent ECS/AUP ports after live collision checks. Configure Coder at `compute.tashan.chat` and workspace apps on a separate wildcard hostname.
- [ ] Terminate TLS at ECS and forward through a monitored AUP-originated tunnel. AUP Coder remains loopback-only.
- [ ] Verify anonymous users cannot access private workspaces/apps. Implement explicit public app policy with a named service and reversible audit record.
- [ ] Test tunnel loss/recovery, Host confusion and same-origin isolation. Commit `feat(network): expose authenticated coder workspaces`.

### Task 8: Accounts, public release and fresh-user acceptance

**Files:** Skill/installer/release metadata, `docs/verification/`, public test scripts.

- [ ] Create the platform owner locally on AUP, then create at least Alice and Bob with temporary passwords. Passwords are stored only in a local system keyring/controlled handoff and never printed in logs or committed.
- [ ] Create personal workspaces and one shared organization workspace; verify isolation and collaboration.
- [ ] Update `tcompute` Release to bundle or securely install the pinned Coder CLI, publish checksummed assets and update the public Skill.
- [ ] A fresh test agent with no repository checkout, Go, Node or Tailscale installs the Skill/CLI, logs in, changes password, SSHes, syncs files, runs all workload classes and accesses private/public apps.
- [ ] Push only after all production and fresh-user gates pass. Record exact SHAs, versions, domains and remaining limitations.

## Self-review

- Coder is the only production account/session authority.
- Incus provides full shell isolation; users never access its daemon.
- Coder Premium organizations/custom roles are not dependencies.
- No host Docker socket, host shell or AUP SSH reaches ordinary users.
- External mutation scripts are help/dry-run by default and have negative self-tests.
- Release is blocked until a real no-Tailscale user journey passes.
