CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE accounts (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    username text NOT NULL UNIQUE,
    password_hash text NOT NULL,
    platform_admin boolean NOT NULL DEFAULT false,
    password_version integer NOT NULL DEFAULT 1 CHECK (password_version > 0),
    must_change_password boolean NOT NULL DEFAULT true,
    disabled_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (username ~ '^[a-z][a-z0-9_.-]{2,63}$')
);

CREATE TABLE devices (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id uuid NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    label text NOT NULL,
    fingerprint_hash bytea NOT NULL,
    first_seen_at timestamptz NOT NULL DEFAULT now(),
    last_seen_at timestamptz NOT NULL DEFAULT now(),
    revoked_at timestamptz,
    UNIQUE (account_id, fingerprint_hash)
);

CREATE TABLE sessions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id uuid NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    device_id uuid NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    refresh_token_hash bytea NOT NULL UNIQUE,
    rotation_family uuid NOT NULL,
    password_version integer NOT NULL CHECK (password_version > 0),
    expires_at timestamptz NOT NULL,
    rotated_at timestamptz,
    revoked_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX sessions_account_active_idx ON sessions (account_id) WHERE revoked_at IS NULL;
CREATE INDEX sessions_device_active_idx ON sessions (device_id) WHERE revoked_at IS NULL;

CREATE TABLE organizations (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name text NOT NULL UNIQUE,
    created_by uuid NOT NULL REFERENCES accounts(id),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE memberships (
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    account_id uuid NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    role text NOT NULL CHECK (role IN ('org_admin', 'developer', 'viewer')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (organization_id, account_id)
);

CREATE TABLE audit_events (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    occurred_at timestamptz NOT NULL DEFAULT now(),
    request_id text NOT NULL,
    actor_account_id uuid REFERENCES accounts(id),
    actor_device_id uuid REFERENCES devices(id),
    effective_role text,
    capability_id text NOT NULL,
    target_type text,
    target_id text,
    outcome text NOT NULL CHECK (outcome IN ('allowed', 'denied', 'failed')),
    source_ip inet,
    user_agent text,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    previous_hash bytea,
    event_hash bytea NOT NULL
);

CREATE FUNCTION reject_audit_event_mutation() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'audit_events are append-only';
END;
$$;

CREATE TRIGGER audit_events_no_update
BEFORE UPDATE OR DELETE ON audit_events
FOR EACH ROW EXECUTE FUNCTION reject_audit_event_mutation();
