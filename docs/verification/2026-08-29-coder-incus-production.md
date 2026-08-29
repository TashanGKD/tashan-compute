# Coder + Incus production verification — 2026-08-29

## Deployed control plane

- Public URL: `https://compute.tashan.chat`
- Coder build API: `v2.35.6+ef299b6`
- PostgreSQL: `17.6-alpine3.22`, pinned digest, loopback `127.0.0.1:55432`
- Coder HTTP/metrics: loopback `127.0.0.1:7080/7081`
- AUP→ECS tunnel: ECS port 13980; ECS firewall rejects non-loopback access
- TLS certificate covers `compute.tashan.chat` and `*.workspaces.compute.tashan.chat`

Public checks returned `health=OK`, dashboard HTTP 200 and a trusted Let's Encrypt certificate. Direct Internet access to ECS port 13980 timed out while ECS loopback returned Coder health.

## Isolation evidence

`deploy/verify-incus-isolation.sh --smoke --host aup-server` passed after live creation and cleanup of a disposable container:

```text
metadata-access=BLOCKED
host-access=BLOCKED
public-egress=PASS
host-mapped-uid=1393216
limits.cpu: "4"
limits.memory: 8GiB
limits.processes: "2048"
security.idmap.isolated: "true"
size: 50GiB
isolation-smoke=PASS
```

The restricted project has a 30 CPU / 48 GiB / 700 GiB aggregate ceiling. Host path allowlists are unset. Nested namespaces are allowed only so the trusted template can provide container builds; no workspace receives the Incus or host Docker socket.

## Workspace and workload evidence

The active template uses prebuilt image fingerprint `ec826a760fb7be23086d5e5c032dab471d147884b3342c75f352c19490bddc10`, an 8 GiB ephemeral root, a 50 GiB personal home, or a 500 GiB platform-created organization home. The image build inputs and guarded rebuild scripts live under `images/workspace/` and `deploy/build-workspace-image.sh`.

Verified through public Coder/tcompute connections:

- root shell inside an unprivileged container;
- Python, Node.js, C, Go and Rust execution;
- PostgreSQL accepting connections and Redis `PONG`;
- Docker-compatible `docker build` via Podman/Buildah;
- stop destroys compute, start recreates it in 8 seconds, persistent marker survives;
- sync dry-run leaves no remote file; `--apply` push/pull round trip is byte-identical;
- `/home/coder/.tcompute/service` restores a resident Python HTTP service after recreation.

## Multi-user evidence

- Created owner, Alice and Bob Coder accounts; credentials are not in Git and temporary passwords are stored only in the local system keychain.
- Alice and Bob each created isolated personal workspaces.
- Alice created `shared-lab`, granted Bob `use`, and both read/wrote its persistent home.
- Bob could not access `alice/alice-personal`.
- Removing Bob automatically restarted `shared-lab`; Bob's next shell was denied. Bob was then re-added for the final collaborative state.
- Admin password reset invalidated two independent Bob sessions with HTTP 401.
- The platform owner created `org-smoke` with Alice as Coder workspace admin; live Incus state showed a 500 GiB organization home and 8 GiB root. Bob's direct attempt to request `space_kind=organization` was rejected during Terraform planning.

## HTTPS service evidence

- Default/private service returned HTTP 303 to an anonymous client.
- Explicit `service public` changed the app to `sharing_level=public`; anonymous HTTPS returned `tcompute-service-pass` at `https://service--shared-lab--alice.workspaces.compute.tashan.chat/`.
- The final state was reverted to private and again returned HTTP 303 anonymously.

## Gates run

- `go test ./...`
- capability manifest/CLI/Skill parity gate and negative self-test
- release metadata parity gate and negative self-test
- public-repository secret scan and negative self-test
- Skill structure validation
- Coder stack, Incus confinement, Docker/Incus firewall, ECS tunnel firewall and public-network self-tests
- Terraform template self-test
- release builder and fresh installer distribution tests

## Independent fresh-user acceptance

A read-only subagent used only the installed Skill and a new temporary HOME/data/bin tree; it did not use repository source or Tailscale.

- Installed public v0.2.0 from the compute edge, then verified the OS-keyring login, 25-entry capability manifest and Bob identity.
- Created a unique personal workspace in 19.40 seconds.
- Public Coder shell, Python, sync dry-run/apply push/pull, persistent marker, stop/start and authorization checks passed.
- An ordinary `FROM alpine:3.22` Docker build initially reproduced Docker Hub timeouts; after adding the template's DaoCloud mirror it pulled, built and ran in 14.64 seconds.
- The marker hash remained identical after stop (12.30 seconds) and start (9.56 seconds).
- Bob could access `alice/shared-lab` but was denied `alice/alice-personal`.
- The unique test workspace was deleted and confirmed absent.
- The test exposed quoted-argument loss in `tcompute shell`; v0.2.1 now POSIX-quotes every remote argv element. Live `python3 -c` and injection-shaped arguments passed after the fix.

Public artifacts are available in GitHub Release `v0.2.1` and at `https://compute.tashan.chat/cli/v0.2.1/`. The installed local Skill was refreshed from merged main and reports version `0.2.1`.

## Known product boundary

Coder OSS does not provide a server-side `must_change_password` flag. The Skill requires temporary-password users to run `tcompute password change` before workspace operations, and administrator resets revoke all existing sessions, but direct Coder Web login cannot yet be hard-blocked until that first change. This remains the principal gap against the stricter account policy.
