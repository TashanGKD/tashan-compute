# Foundation, Authentication, and Distribution Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the independently deployable Tashan Compute foundation in which a public Skill installs a safe `tcompute` CLI, while only server-created accounts and server-stored roles can access protected or administrative APIs.

**Architecture:** A Go monorepo produces separate `tcompute`, `tcompute-api`, and `tcompute-admin` binaries. The API uses PostgreSQL as the authorization truth, server-held Ed25519 keys for short-lived access tokens, rotating opaque refresh tokens bound to devices, and a capability manifest that CI compares with CLI and Skill bindings. The public installer never receives credentials; the first platform administrator is bootstrapped only by running `tcompute-admin` on the server with direct database access.

**Tech Stack:** Go 1.26, PostgreSQL 17, Redis 8, `net/http`, `github.com/jackc/pgx/v5`, `github.com/spf13/cobra`, `github.com/golang-jwt/jwt/v5`, `golang.org/x/crypto/argon2`, Testcontainers for Go, GitHub Actions, POSIX shell installer.

---

## Scope and plan boundaries

This plan delivers the secure identity and distribution foundation only. It intentionally does not create file, space, snapshot, Secret, Executor, BuildKit, service, database-workload, or Gateway behavior. Those capabilities receive their own plans after this foundation exposes stable identity, authorization, audit and capability contracts.

Security rejection cases that must exist before happy-path implementation:

1. A modified public CLI sends `role=platform_admin`, a self-signed JWT, or an unsigned JWT and every admin route returns `401` or `403`.
2. A valid ordinary user, a disabled user, a revoked device, an expired token, and a user still required to change the initial password cannot call administrative or future space APIs.
3. Passwords, initial passwords, refresh tokens and private signing keys supplied as command-line arguments are rejected; sensitive values are accepted only through a hidden TTY prompt or stdin and never appear in output or audit bodies.
4. Running any binary with no arguments is read-only: it prints help and does not access the network, database, keychain or filesystem outside process-local state.

## Target file map

```text
cmd/
  tcompute/main.go                 public CLI entrypoint
  tcompute-api/main.go             HTTP API entrypoint
  tcompute-admin/main.go           server-local bootstrap and recovery entrypoint
internal/
  audit/                           append-only security events and redaction
  auth/                            password, token, session and first-login rules
  capability/                      manifest schema and registry
  cli/                             Cobra commands, output and safe prompts
  config/                          explicit local/production configuration
  httpapi/                         router, middleware, error envelope and handlers
  identity/                        account, device, organization and role domain types
  store/postgres/                  repositories and migrations
  testkit/                         PostgreSQL and HTTP integration fixtures
migrations/                        ordered SQL migrations
capabilities/manifest.json         single capability source of truth
skill/tashan-compute/              public Codex Skill and fixed-version installer
release/cli-release.json           CLI release source of truth
scripts/                           consistency gates and negative self-tests
tests/distribution/                fresh-user install and secret-leak tests
.github/workflows/ci.yml           pull-request verification
```

### Task 1: Bootstrap the Go module and safe binary defaults

**Files:**
- Create: `go.mod`
- Create: `cmd/tcompute/main.go`
- Create: `cmd/tcompute-api/main.go`
- Create: `cmd/tcompute-admin/main.go`
- Create: `internal/cli/root.go`
- Create: `internal/cli/root_test.go`
- Create: `internal/buildinfo/version.go`
- Modify: `.gitignore`

- [x] **Step 1: Write the failing no-argument safety test**

```go
func TestRootWithoutArgumentsOnlyPrintsHelp(t *testing.T) {
	called := false
	cmd := NewRoot(Dependencies{NetworkProbe: func() { called = true }})
	cmd.SetArgs(nil)
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := cmd.Execute()

	require.NoError(t, err)
	require.False(t, called)
	require.Contains(t, stdout.String(), "Tashan Compute")
	require.Empty(t, stderr.String())
}
```

- [x] **Step 2: Run the test and verify RED**

Run: `go test ./internal/cli -run TestRootWithoutArgumentsOnlyPrintsHelp -v`

Expected: FAIL because `NewRoot` and `Dependencies` do not exist.

- [x] **Step 3: Implement the minimal root command**

```go
type Dependencies struct {
	NetworkProbe func()
}

func NewRoot(_ Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "tcompute",
		Short:         "Tashan Compute CLI",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.Version = buildinfo.Version
	return cmd
}
```

Each `main.go` must only construct its root and call `Execute`; it must not read configuration before Cobra selects a subcommand.

- [x] **Step 4: Verify GREEN and all binaries build**

Run: `go test ./internal/cli -run TestRootWithoutArgumentsOnlyPrintsHelp -v && go build ./cmd/...`

Expected: PASS and three binaries compile.

- [x] **Step 5: Commit**

```bash
git add go.mod cmd internal/cli internal/buildinfo .gitignore
git commit -m "feat(cli): add safe command foundation"
```

### Task 2: Establish the capability manifest and executable drift gate

**Files:**
- Create: `capabilities/manifest.json`
- Create: `internal/capability/manifest.go`
- Create: `internal/capability/manifest_test.go`
- Create: `internal/cli/bindings.json`
- Create: `skill/tashan-compute/capability-references.json`
- Create: `scripts/check-capability-coverage/main.go`
- Create: `scripts/check-capability-coverage.self-test.sh`

- [x] **Step 1: Write the failing manifest validation tests**

```go
func TestManifestRejectsDuplicateIDsAndMissingCLIBindings(t *testing.T) {
	duplicate := []Capability{{ID: "system.health.read"}, {ID: "system.health.read"}}
	require.ErrorIs(t, Validate(duplicate), ErrDuplicateCapability)

	manifest := []Capability{{ID: "admin.user.create", CLI: "admin user create"}}
	require.Error(t, CheckBindings(manifest, map[string]string{}))
}
```

- [x] **Step 2: Run the test and verify RED**

Run: `go test ./internal/capability -v`

Expected: FAIL because the package does not exist.

- [x] **Step 3: Define the foundation capabilities**

The JSON manifest must contain exactly these initial IDs and CLI bindings:

```json
[
  {"id":"system.health.read","version":1,"cli":"health","auth":"public","side_effect":"none"},
  {"id":"capability.list","version":1,"cli":"capability list","auth":"public","side_effect":"none"},
  {"id":"auth.login","version":1,"cli":"auth login","auth":"public","side_effect":"session"},
  {"id":"auth.password.initial_change","version":1,"cli":"auth password initial-change","auth":"initial_password","side_effect":"session"},
  {"id":"auth.password.change","version":1,"cli":"auth password change","auth":"user","side_effect":"session"},
  {"id":"auth.refresh","version":1,"cli":"auth refresh","auth":"refresh_token","side_effect":"session"},
  {"id":"auth.logout","version":1,"cli":"auth logout","auth":"user","side_effect":"revoke"},
  {"id":"auth.whoami","version":1,"cli":"auth whoami","auth":"user","side_effect":"none"},
  {"id":"device.list","version":1,"cli":"device list","auth":"user","side_effect":"none"},
  {"id":"device.revoke","version":1,"cli":"device revoke","auth":"user","side_effect":"revoke"},
  {"id":"admin.user.create","version":1,"cli":"admin user create","auth":"platform_admin","side_effect":"write"},
  {"id":"admin.user.reset_password","version":1,"cli":"admin user reset-password","auth":"platform_admin","side_effect":"revoke"},
  {"id":"admin.user.disable","version":1,"cli":"admin user disable","auth":"platform_admin","side_effect":"revoke"},
  {"id":"admin.organization.create","version":1,"cli":"admin org create","auth":"platform_admin","side_effect":"write"},
  {"id":"organization.list","version":1,"cli":"org list","auth":"user","side_effect":"none"},
  {"id":"organization.member.add","version":1,"cli":"org member add","auth":"org_admin","side_effect":"write"},
  {"id":"organization.member.remove","version":1,"cli":"org member remove","auth":"org_admin","side_effect":"revoke"},
  {"id":"organization.member.role_set","version":1,"cli":"org member role-set","auth":"org_admin","side_effect":"write"},
  {"id":"audit.list","version":1,"cli":"audit list","auth":"scoped_auditor","side_effect":"none"}
]
```

The Go validator must reject empty IDs, duplicates, unknown auth classes, unknown side effects and missing CLI bindings.

- [x] **Step 4: Implement and run the gate negative self-test**

`check-capability-coverage.self-test.sh` copies fixtures to a temporary directory, removes `admin.user.create` from CLI bindings, runs the gate, and asserts a non-zero exit plus the exact message `missing CLI binding: admin.user.create`. It then changes one Skill ID to `admin.user.creat` and asserts `unknown Skill capability: admin.user.creat`.

Run: `go test ./internal/capability -v && bash scripts/check-capability-coverage.self-test.sh`

Expected: unit tests PASS and both pathological fixtures are caught.

- [x] **Step 5: Commit**

```bash
git add capabilities internal/capability internal/cli/bindings.json skill/tashan-compute/capability-references.json scripts
git commit -m "feat(capability): define foundation registry"
```

### Task 3: Add fail-closed configuration and HTTP error contracts

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`
- Create: `internal/httpapi/errors.go`
- Create: `internal/httpapi/server.go`
- Create: `internal/httpapi/server_test.go`
- Create: `.env.example`

- [x] **Step 1: Write failing production-configuration tests**

```go
func TestProductionRejectsMissingSecretsAndPlainHTTP(t *testing.T) {
	_, err := Load(MapSource{"TCOMPUTE_ENV": "production", "TCOMPUTE_PUBLIC_URL": "http://compute.example.test"})
	require.ErrorContains(t, err, "production public URL must use https")

	_, err = Load(MapSource{"TCOMPUTE_ENV": "production", "TCOMPUTE_PUBLIC_URL": "https://compute.example.test"})
	require.ErrorContains(t, err, "TCOMPUTE_DATABASE_URL is required")
}

func TestDevelopmentDefaultsStayOnLoopback(t *testing.T) {
	cfg, err := Load(MapSource{})
	require.NoError(t, err)
	require.Equal(t, "127.0.0.1:8180", cfg.ListenAddress)
	require.Equal(t, "http://127.0.0.1:8180", cfg.PublicURL)
}
```

- [x] **Step 2: Run the tests and verify RED**

Run: `go test ./internal/config ./internal/httpapi -v`

Expected: FAIL because both packages are missing.

- [x] **Step 3: Implement explicit configuration and stable errors**

Define `Config` with `Environment`, `ListenAddress`, `PublicURL`, `DatabaseURL`, `RedisURL`, `AccessPrivateKeyFile`, `AccessPublicKeyFile`, `RefreshPepperFile`, `TrustedProxyCIDRs` and `AllowedOrigins`. Production rejects loopback data services, wildcard origins, obvious dummy values, missing key files and non-HTTPS public URLs.

The error envelope must be:

```go
type ErrorResponse struct {
	Error struct {
		Code      string            `json:"code"`
		Message   string            `json:"message"`
		RequestID string            `json:"request_id"`
		Fields    map[string]string `json:"fields,omitempty"`
	} `json:"error"`
}
```

`GET /v1/health` returns only build version and status; it never exposes dependency URLs, hostnames or environment values.

- [x] **Step 4: Verify GREEN**

Run: `go test ./internal/config ./internal/httpapi -v`

Expected: PASS, including a test that malformed JSON returns `request.invalid_json` and no Go error text.

- [x] **Step 5: Commit**

```bash
git add internal/config internal/httpapi .env.example
git commit -m "feat(api): add fail-closed configuration"
```

### Task 4: Create the PostgreSQL identity schema and migration gate

**Files:**
- Create: `migrations/0001_identity.sql`
- Create: `internal/store/postgres/migrate.go`
- Create: `internal/store/postgres/migrate_test.go`
- Create: `internal/testkit/postgres.go`
- Create: `compose.test.yml`

- [x] **Step 1: Write the failing schema integration test**

```go
func TestIdentitySchemaEnforcesServerRolesAndSessionRevocation(t *testing.T) {
	db := testkit.Postgres(t)
	require.NoError(t, postgres.Migrate(context.Background(), db))

	for _, table := range []string{"accounts", "devices", "sessions", "organizations", "memberships", "audit_events"} {
		require.True(t, tableExists(t, db, table), table)
	}
	require.Error(t, insertMembershipWithRole(t, db, "client_supplied_admin"))
}
```

- [x] **Step 2: Run the test and verify RED**

Run: `go test ./internal/store/postgres -run TestIdentitySchema -v`

Expected: FAIL because the migration runner is missing.

- [x] **Step 3: Implement the ordered migration**

`0001_identity.sql` must create UUID-keyed accounts, devices, sessions, organizations, memberships and append-only audit events. Role columns use PostgreSQL CHECK constraints with only `platform_admin`, `org_admin`, `developer`, and `viewer`. Accounts include `password_version`, `must_change_password`, `disabled_at`; sessions include a hashed refresh token, `device_id`, expiry, rotation family and `revoked_at`.

No migration may seed usernames, passwords or signing keys. The migration runner uses a PostgreSQL advisory lock and records SHA-256 for each applied file; a changed applied migration fails closed.

- [x] **Step 4: Run schema and migration-pathology tests**

Run: `docker compose -f compose.test.yml up -d postgres && go test ./internal/store/postgres -v`

Expected: PASS for first and repeated migration; PASS when the test proves an altered applied migration is rejected.

- [x] **Step 5: Commit**

```bash
git add migrations internal/store/postgres internal/testkit compose.test.yml
git commit -m "feat(storage): add identity schema"
```

### Task 5: Implement password, initial-login and account-state rules

**Files:**
- Create: `internal/auth/password.go`
- Create: `internal/auth/password_test.go`
- Create: `internal/auth/service.go`
- Create: `internal/auth/service_test.go`
- Create: `internal/identity/account.go`
- Create: `internal/store/postgres/accounts.go`

- [x] **Step 1: Write the adversarial password tests first**

```go
func TestInitialPasswordCannotUseProtectedCapabilities(t *testing.T) {
	svc, account := fixtureMustChangePassword(t)
	decision := svc.Authorize(account, "admin.user.create")
	require.Equal(t, auth.DenyInitialPassword, decision)
}

func TestClientCannotChooseRoleWhenAccountIsCreated(t *testing.T) {
	svc := fixtureAdminService(t)
	created, err := svc.CreateAccount(context.Background(), auth.CreateAccountInput{
		Username: "alice", InitialPassword: "correct horse battery staple",
	})
	require.NoError(t, err)
	require.False(t, created.PlatformAdmin)
}
```

Also test empty/oversized usernames, Unicode-confusable usernames, a password equal to the username, overlong passwords, wrong-password timing tolerance and disabled accounts.

- [x] **Step 2: Run the tests and verify RED**

Run: `go test ./internal/auth -run 'Password|Initial|ClientCannotChooseRole' -v`

Expected: FAIL because the service is missing.

- [x] **Step 3: Implement Argon2id and account state transitions**

Use random 16-byte salts and PHC-format Argon2id hashes with parameters stored in the hash. `CreateAccountInput` has no role field. Only the server-local bootstrap path can set the first `platform_admin`; subsequent role assignment is a separate audited server authorization operation.

Initial-password change increments `password_version`, clears `must_change_password`, revokes every session, and returns no session until the user performs a fresh login with the new password.

- [x] **Step 4: Verify GREEN**

Run: `go test ./internal/auth -v`

Expected: PASS and no test logs contain any fixture password.

- [x] **Step 5: Commit**

```bash
git add internal/auth internal/identity internal/store/postgres/accounts.go
git commit -m "feat(auth): enforce managed accounts"
```

### Task 6: Implement server-signed access tokens and rotating device sessions

**Files:**
- Create: `internal/auth/access_token.go`
- Create: `internal/auth/access_token_test.go`
- Create: `internal/auth/session.go`
- Create: `internal/auth/session_test.go`
- Create: `internal/store/postgres/sessions.go`

- [x] **Step 1: Write token forgery and revocation tests before implementation**

```go
func TestRejectsSelfSignedAndClientRoleTokens(t *testing.T) {
	verifier := fixtureVerifier(t)
	attacker := newEd25519Key(t)
	token := signClaims(t, attacker, Claims{Subject: "alice", Role: "platform_admin"})
	_, err := verifier.Verify(context.Background(), token)
	require.ErrorIs(t, err, ErrInvalidSignature)
}

func TestPasswordResetRevokesEveryDevice(t *testing.T) {
	svc, account, sessions := fixtureTwoDeviceSessions(t)
	require.NoError(t, svc.AdminResetPassword(context.Background(), account.ID, "new initial password"))
	for _, token := range sessions {
		_, err := svc.Refresh(context.Background(), token)
		require.ErrorIs(t, err, ErrSessionRevoked)
	}
}
```

Also test expired access tokens, refresh replay after rotation, mismatched device IDs, disabled accounts and old `password_version`.

- [x] **Step 2: Run tests and verify RED**

Run: `go test ./internal/auth -run 'Token|Session|Reset|Refresh' -v`

Expected: FAIL because token/session types are missing.

- [x] **Step 3: Implement session binding**

Access tokens use Ed25519, a fixed issuer and audience, a maximum 10-minute lifetime, `sub`, `sid`, `did`, `password_version`, `iat`, `nbf`, `exp`, and no trusted role claim. Middleware loads the session and current account/membership roles from PostgreSQL for protected operations.

Refresh tokens are 32 random bytes returned once, stored only as a pepper-keyed hash, rotated on every use and grouped into a family. Replay revokes the family. Device revoke, account disable and password reset revoke matching sessions transactionally.

- [x] **Step 4: Verify GREEN**

Run: `go test ./internal/auth ./internal/store/postgres -v`

Expected: PASS, including replay and all-device reset cases.

- [x] **Step 5: Commit**

```bash
git add internal/auth internal/store/postgres/sessions.go
git commit -m "security(auth): bind tokens to devices"
```

### Task 7: Add authenticated API middleware, admin routes and audit redaction

**Files:**
- Create: `internal/httpapi/auth_middleware.go`
- Create: `internal/httpapi/auth_handlers.go`
- Create: `internal/httpapi/admin_handlers.go`
- Create: `internal/httpapi/organization_handlers.go`
- Create: `internal/httpapi/admin_integration_test.go`
- Create: `internal/audit/event.go`
- Create: `internal/audit/redact.go`
- Create: `internal/audit/redact_test.go`
- Create: `internal/store/postgres/audit.go`

- [x] **Step 1: Write the real HTTP rejection matrix**

```go
func TestAdminRoutesRejectEveryClientSideEscalation(t *testing.T) {
	api := testkit.API(t)
	cases := []requestCase{
		{name: "no token", token: "", want: http.StatusUnauthorized},
		{name: "self signed", token: fixtureSelfSignedAdminJWT(t), want: http.StatusUnauthorized},
		{name: "ordinary user", token: api.Login(t, "alice"), want: http.StatusForbidden},
		{name: "revoked device", token: api.RevokedDeviceToken(t), want: http.StatusUnauthorized},
		{name: "initial password", token: api.InitialPasswordToken(t), want: http.StatusForbidden},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := api.Post(t, "/v1/admin/users", tc.token, `{"username":"bob","initial_password":"ignored"}`)
			require.Equal(t, tc.want, res.StatusCode)
		})
	}
}
```

- [x] **Step 2: Run the integration test and verify RED**

Run: `go test ./internal/httpapi -run TestAdminRoutesRejectEveryClientSideEscalation -v`

Expected: FAIL because middleware and routes do not exist.

- [x] **Step 3: Implement routes with server-side authorization**

Register the paths listed in Task 2. The request body for account creation contains only `username`; the initial password is carried in a separately parsed sensitive field that is redacted before request logging and never stored in idempotency response bodies. Admin reset returns only account ID, `must_change_password=true`, and revoked-device count.

The audit event records actor account, device, effective server role, request ID, trusted source IP, user agent, capability, target type/ID, result and redacted metadata. Redaction recursively removes keys matching password, token, authorization, cookie, secret and private-key patterns.

- [x] **Step 4: Verify success and leakage rejection**

Run: `go test ./internal/httpapi ./internal/audit -v`

Expected: PASS. A dedicated test serializes API logs and audit rows and proves they contain none of the fixture password, refresh token, bearer token or private key bytes.

- [x] **Step 5: Commit**

```bash
git add internal/httpapi internal/audit internal/store/postgres/audit.go
git commit -m "feat(api): enforce server-side admin roles"
```

### Task 8: Add the server-local administrator bootstrap command

**Files:**
- Create: `internal/admin/root.go`
- Create: `internal/admin/bootstrap.go`
- Create: `internal/admin/bootstrap_test.go`
- Modify: `cmd/tcompute-admin/main.go`

- [x] **Step 1: Write bootstrap safety tests**

```go
func TestBootstrapRequiresDirectDatabaseAndHiddenPassword(t *testing.T) {
	cmd := NewRoot(Dependencies{IsTerminal: func() bool { return false }})
	cmd.SetArgs([]string{"bootstrap", "--username", "root", "--password", "leaked"})
	err := cmd.Execute()
	require.ErrorContains(t, err, "unknown flag: --password")

	cmd = NewRoot(Dependencies{IsTerminal: func() bool { return false }})
	cmd.SetArgs([]string{"bootstrap", "--username", "root"})
	err = cmd.Execute()
	require.ErrorContains(t, err, "interactive terminal or --password-stdin is required")
}
```

- [x] **Step 2: Run the test and verify RED**

Run: `go test ./internal/admin -v`

Expected: FAIL because the package does not exist.

- [x] **Step 3: Implement one-time bootstrap**

`tcompute-admin bootstrap --username <name>` reads the password twice from a hidden TTY. `--password-stdin` is allowed for controlled automation; no password flag or environment variable exists. The command requires a direct PostgreSQL connection, refuses HTTP API URLs, takes an advisory lock, and succeeds only when no platform administrator exists. A second attempt returns `bootstrap.already_completed` without changing any account.

- [x] **Step 4: Verify GREEN and command help**

Run: `go test ./internal/admin -v && go run ./cmd/tcompute-admin --help`

Expected: PASS; help contains no password value flag and no remote bootstrap option.

- [x] **Step 5: Commit**

```bash
git add cmd/tcompute-admin internal/admin
git commit -m "security(admin): add local bootstrap command"
```

### Task 9: Implement CLI authentication, credentials and admin commands

**Files:**
- Create: `internal/client/client.go`
- Create: `internal/client/client_test.go`
- Create: `internal/cli/auth.go`
- Create: `internal/cli/admin.go`
- Create: `internal/cli/device.go`
- Create: `internal/cli/organization.go`
- Create: `internal/cli/output.go`
- Create: `internal/credentials/store.go`
- Create: `internal/credentials/macos_keychain.go`
- Create: `internal/credentials/linux_secret_service.go`
- Create: `internal/credentials/memory.go`
- Create: `internal/credentials/store_test.go`

- [x] **Step 1: Write CLI secret-input and role-forgery tests**

```go
func TestLoginRejectsPasswordArgumentAndNeverPrintsTokens(t *testing.T) {
	cmd := NewRoot(fixtureCLI(t))
	cmd.SetArgs([]string{"auth", "login", "--username", "alice", "--password", "leaked"})
	err := cmd.Execute()
	require.ErrorContains(t, err, "unknown flag: --password")

	stdout, stderr := executeWithStdin(t, []string{"auth", "login", "--username", "alice", "--password-stdin"}, "fixture-password\n")
	require.NotContains(t, stdout+stderr, "fixture-password")
	require.NotContains(t, stdout+stderr, "fixture-refresh-token")
}
```

Add a test invoking `admin user create --role platform_admin` and assert `unknown flag: --role`; public CLI account creation cannot choose platform role.

- [x] **Step 2: Run CLI tests and verify RED**

Run: `go test ./internal/cli ./internal/credentials ./internal/client -v`

Expected: FAIL because the commands and stores are missing.

- [x] **Step 3: Implement secure credential behavior**

macOS uses Keychain and Linux uses Secret Service. When secure storage is unavailable, interactive login holds credentials in memory and prints a clear non-persistent warning; it does not silently create a plaintext file. Every command supports stable JSON output, keeps stdout machine-readable and sends warnings to stderr.

Implement every foundation binding from Task 2. `admin user create` and `reset-password` accept initial passwords only through hidden confirmation or `--password-stdin`. The client never sends a role claim as an authentication authority; organization role changes target only `org_admin`, `developer`, or `viewer` and require a server-authorized caller.

- [x] **Step 4: Verify GREEN and help safety**

Run: `go test ./internal/cli ./internal/credentials ./internal/client -v && go run ./cmd/tcompute --help`

Expected: PASS; help documents that installation does not grant an account and contains no AUP SSH instructions.

- [x] **Step 5: Commit**

```bash
git add internal/client internal/cli internal/credentials
git commit -m "feat(cli): add managed account commands"
```

### Task 10: Build the public Skill, fixed-version installer and release contract

**Files:**
- Create: `skill/tashan-compute/SKILL.md`
- Create: `skill/tashan-compute/agents/openai.yaml`
- Create: `skill/tashan-compute/scripts/install-cli.sh`
- Create: `skill/tashan-compute/release.json`
- Create: `skill/tashan-compute/references/authentication.md`
- Create: `skill/tashan-compute/references/security.md`
- Create: `release/cli-release.json`
- Create: `scripts/build-cli-release.sh`
- Create: `scripts/check-release-contract/main.go`
- Create: `scripts/check-release-contract.self-test.sh`
- Create: `tests/distribution/install-cli.sh`

- [x] **Step 1: Write installer adversarial tests before the installer**

The test must construct and reject all of these inputs:

```text
--version ../../tmp/pwn
unsupported OS/CPU
SHA-256 mismatch
checksum file without an exact asset line
archive entry with /absolute, ../escape, extra top-level directory, symlink or hardlink
existing tcompute executable not managed by this installer
failed smoke test after extraction
```

Each failure test asserts the previous installed version still runs and no partial target directory remains.

- [x] **Step 2: Run installer tests and verify RED**

Run: `bash tests/distribution/install-cli.sh`

Expected: FAIL because the installer is missing.

- [x] **Step 3: Implement safe fixed-version distribution**

The installer with no arguments prints help and performs no network or writes. `--install` reads the exact version and assets from `release.json`, downloads only from `TashanGKD/tashan-compute` GitHub Releases, verifies SHA-256 and archive layout, installs under `${XDG_DATA_HOME:-$HOME/.local/share}/tcompute/versions/<version>`, and atomically updates `${TCOMPUTE_BIN_DIR:-$HOME/.local/bin}/tcompute` only after `--version` and no-argument smoke tests pass.

Release archives include the Go binary, license, version metadata and no configuration or credentials. Skill documentation says explicitly: installing the public Skill/CLI does not create an account or grant AUP/admin access.

- [x] **Step 4: Run distribution and release drift tests**

Run: `bash tests/distribution/install-cli.sh && bash scripts/check-release-contract.self-test.sh`

Expected: PASS; negative release fixture fails with the exact drift message and the prior install survives every injected failure.

- [x] **Step 5: Commit**

```bash
git add skill release scripts/build-cli-release.sh scripts/check-release-contract* tests/distribution
git commit -m "feat(distribution): add public skill installer"
```

### Task 11: Wire CI, secret scanning and real identity E2E

**Files:**
- Create: `.github/workflows/ci.yml`
- Create: `scripts/verify-foundation.sh`
- Create: `scripts/verify-foundation.self-test.sh`
- Create: `scripts/check-public-repo-secrets.sh`
- Create: `scripts/check-public-repo-secrets.self-test.sh`
- Create: `tests/e2e/identity_lifecycle_test.go`
- Create: `docs/verification/foundation-template.md`
- Modify: `README.md`

- [x] **Step 1: Write gate self-tests before the gates**

`check-public-repo-secrets.self-test.sh` builds a temporary Git fixture containing representative private-key headers, bearer tokens, database URLs with passwords, password assignments and AUP SSH host/user fields. It asserts the gate rejects every fixture, then replaces each value with documented redacted examples and asserts success.

`verify-foundation.self-test.sh` removes one gate from a copied verification script and asserts the verifier fails with `required gate missing`, proving the aggregator cannot silently skip a security gate.

- [x] **Step 2: Run the self-tests and verify RED**

Run: `bash scripts/check-public-repo-secrets.self-test.sh && bash scripts/verify-foundation.self-test.sh`

Expected: FAIL because the scripts do not exist.

- [x] **Step 3: Implement CI and two-user lifecycle E2E**

`identity_lifecycle_test.go` must start real PostgreSQL and Redis, bootstrap one admin through the server-local command, create two ordinary accounts and one organization through the HTTPS test API, force both initial password changes, log in from separate devices, add both users to the organization, revoke one device, reset one password and verify every old session is rejected.

The same test must send self-signed and client-role JWTs to `/v1/admin/users` and assert rejection, then use the genuine admin session and assert success. Captured stdout, stderr, API logs and audit rows are scanned for fixture passwords, tokens and key bytes.

CI runs formatting, `go vet`, unit tests, PostgreSQL/Redis integration tests, gate self-tests, distribution tests, `go test -race`, and the two-user E2E. Release publication is a separate manually approved workflow and is not added until a real production health gate exists.

- [x] **Step 4: Run the full verifier**

Run: `bash scripts/verify-foundation.sh`

Expected: all checks PASS from a clean clone with only Docker and Go installed. No command connects to production or sends credentials.

- [x] **Step 5: Update README with only verified commands and commit**

```bash
git add .github scripts tests/e2e docs/verification README.md
git commit -m "ci(security): enforce public repo boundaries"
```

## Final plan self-review checklist

- Foundation spec coverage: public distribution, managed accounts, first-login password change, reset revocation, device sessions, platform-admin-only user/org creation, organization roles, audit, Capability/CLI/Skill parity and secret-free repository are assigned to concrete tasks.
- Deliberately deferred: space/file/storage, snapshot implementation beyond the identity contract, workload execution, BuildKit, long-running services, database workloads and dynamic Gateway.
- Type consistency: capability IDs and CLI bindings in Task 2 are the names used by API/CLI tests in later tasks; platform role is never accepted from account-create input.
- Safe defaults: all no-argument binaries are help-only; source defaults are loopback; bootstrap is direct-DB-only; installer is help-only without `--install`; no release workflow can publish automatically.
- Negative evidence: every new gate has a same-named self-test and every security boundary has an HTTP or CLI rejection test before happy-path implementation.
