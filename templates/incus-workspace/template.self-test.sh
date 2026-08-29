#!/usr/bin/env bash
set -euo pipefail

template_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
main="$template_dir/main.tf"

for required in \
  'source  = "lxc/incus"' \
  'version = "1.2.0"' \
  'source  = "coder/coder"' \
  'version = "2.18.0"' \
  'resource "incus_instance" "workspace"' \
  'count = data.coder_workspace.me.start_count' \
  'resource "incus_storage_volume" "home"' \
  'resource "coder_app" "service"' \
  'default      = "owner"' \
  'value = "public"' \
  'subdomain    = true' \
  '"50GiB"' \
  '500GiB' \
  'space_kind' \
  'data.coder_workspace_owner.me.name == "tashan-admin"' \
  'path   = "/home/coder"' \
  'size = "8GiB"' \
  'ec826a760fb7be23086d5e5c032dab471d147884b3342c75f352c19490bddc10' \
  '"security.privileged"' \
  '"security.nesting"' \
  '"security.idmap.isolated"' \
  '["default"]' \
  '"user-955"' \
  'ExecStart=/usr/local/bin/coder agent' \
  'CODER_AGENT_URL=https://compute.tashan.chat' \
  'content     = coder_agent.main.token' \
  'User=root' \
  'CODER_AGENT_TOKEN_FILE=/etc/tcompute-agent-token' \
  'ConditionPathIsExecutable=/home/coder/.tcompute/service' \
  'Restart=always' \
  'target_path = "/etc/tcompute-agent-token"' \
  'mode        = "0600"' \
  'resource_id = incus_instance.workspace[0].name' \
  'command = ["/usr/local/sbin/tcompute-boundary-check"]'; do
  grep -Fq "$required" "$main" || {
    echo "template missing invariant: $required" >&2
    exit 1
  }
done

if grep -Fq 'coder_agent.main.init_script' "$main"; then
  echo 'workspace restarts would re-download the Coder agent' >&2
  exit 1
fi

grep -Eq '"security\.nesting"[[:space:]]*=[[:space:]]*"true"' "$main" || {
  echo 'container-build workspace does not enable nested namespaces' >&2
  exit 1
}

if grep -Eiq '(security\.privileged[^f]*true|source[[:space:]]*=[[:space:]]*"/)' "$main"; then
  echo 'template contains a privileged, nested, socket, or host-path escape' >&2
  exit 1
fi

echo 'incus-workspace template self-test: PASS'
