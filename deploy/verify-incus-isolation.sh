#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  verify-incus-isolation.sh --smoke --host aup-server

With no arguments this command only prints help. The smoke mode creates and
always deletes one fixed-name disposable container.
EOF
}

mode=help
host=
while (($#)); do
  case "$1" in
    --smoke)
      [[ "$mode" == help ]] || { echo 'choose exactly one mode' >&2; exit 2; }
      mode=smoke
      shift
      ;;
    --host)
      [[ $# -ge 2 ]] || { echo '--host requires a value' >&2; exit 2; }
      host=$2
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      exit 2
      ;;
  esac
done

if [[ "$mode" == help ]]; then
  usage
  exit 0
fi
[[ "$host" == aup-server ]] || { echo 'refusing: --host must be aup-server' >&2; exit 1; }

ssh_bin=${TCOMPUTE_SSH_BIN:-ssh}
remote_hostname=$($ssh_bin "$host" hostname | tr -d '\r\n')
[[ "$remote_hostname" == aup-test-01 ]] || {
  echo "refusing unexpected remote hostname: $remote_hostname" >&2
  exit 1
}

$ssh_bin -tt "$host" 'set -eu
name=tcompute-isolation-smoke
cleanup() {
  sudo -u tcompute -H incus delete -f "$name" >/dev/null 2>&1 || true
}
trap cleanup EXIT INT TERM
cleanup

uid=$(id -u tcompute)
project="user-$uid"
network="incusbr-$uid"
project_config=$(sudo incus project show "$project")
grep -F '\''restricted.containers.nesting: allow'\'' <<EOF
$project_config
EOF
grep -F '\''restricted.devices.disk: allow'\'' <<EOF
$project_config
EOF
if grep -Fq '\''restricted.devices.disk.paths:'\'' <<EOF
$project_config
EOF
then
  echo host-disk-path-allowlist=FAIL >&2
  exit 1
fi
grep -F '\''restricted.devices.gpu: block'\'' <<EOF
$project_config
EOF
network_config=$(sudo incus network show "$network" --project default)
grep -F '\''security.acls: tcompute-egress'\'' <<EOF
$network_config
EOF
grep -F '\''security.acls.default.egress.action: allow'\'' <<EOF
$network_config
EOF

sudo -u tcompute -H incus launch images:ubuntu/24.04 "$name"
ready=0
for attempt in $(seq 1 60); do
  if sudo -u tcompute -H incus exec "$name" -- sh -lc '\''test -e /etc/resolv.conf && ip -4 route | grep -q default'\'' 2>/dev/null; then
    ready=1
    break
  fi
  sleep 1
done
test "$ready" = 1

sudo -u tcompute -H incus exec "$name" -- sh -euc '\''
test "$(id -u)" = 0
test ! -S /var/lib/incus/unix.socket
getent ahostsv4 example.com >/dev/null
timeout 8 bash -c "exec 3<>/dev/tcp/example.com/443"
if timeout 4 bash -c "exec 3<>/dev/tcp/169.254.169.254/80" 2>/dev/null; then
  echo metadata-access=FAIL >&2
  exit 1
fi
echo metadata-access=BLOCKED
if timeout 4 bash -c "exec 3<>/dev/tcp/100.126.44.113/22" 2>/dev/null; then
  echo host-access=FAIL >&2
  exit 1
fi
echo host-access=BLOCKED
echo public-egress=PASS
'\''

pid=$(sudo -u tcompute -H incus info "$name" | awk '\''/PID:/ {print $2}'\'')
test -n "$pid"
host_uid=$(ps -o uid= -p "$pid" | tr -d " ")
test -n "$host_uid"
test "$host_uid" != 0
echo "host-mapped-uid=$host_uid"

expanded=$(sudo -u tcompute -H incus config show "$name" --expanded)
grep -F '\''limits.cpu: "4"'\'' <<EOF
$expanded
EOF
grep -F '\''limits.memory: 8GiB'\'' <<EOF
$expanded
EOF
grep -F '\''limits.processes: "2048"'\'' <<EOF
$expanded
EOF
grep -F '\''security.idmap.isolated: "true"'\'' <<EOF
$expanded
EOF
if grep -Fq '\''security.nesting: "true"'\'' <<EOF
$expanded
EOF
then
  echo default-workspace-nesting=FAIL >&2
  exit 1
fi
grep -F '\''size: 50GiB'\'' <<EOF
$expanded
EOF

cleanup
trap - EXIT INT TERM
echo isolation-smoke=PASS
'

echo 'Incus live isolation verification completed'
