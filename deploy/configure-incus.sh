#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  configure-incus.sh --check --host aup-server
  configure-incus.sh --apply --host aup-server

With no arguments this command only prints help. --apply changes only the
dedicated tcompute restricted Incus project and its managed bridge ACL.
EOF
}

mode=help
host=
while (($#)); do
  case "$1" in
    --check|--apply)
      [[ "$mode" == help ]] || { echo 'choose exactly one mode' >&2; exit 2; }
      mode=${1#--}
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

if [[ "$mode" == check ]]; then
  $ssh_bin -tt "$host" 'set -eu
uid=$(id -u tcompute)
project="user-$uid"
network="incusbr-$uid"
sudo incus project show "$project"
sudo incus profile show default --project "$project"
sudo incus network show "$network" --project default
sudo incus network acl show tcompute-egress --project default
'
  exit 0
fi

$ssh_bin -tt "$host" 'set -eu
uid=$(id -u tcompute)
project="user-$uid"
network="incusbr-$uid"

sudo incus project set "$project" \
  restricted=true \
  restricted.containers.nesting=allow \
  restricted.devices.disk=allow \
  restricted.devices.gpu=block \
  "restricted.networks.access=$network" \
  limits.containers=20 \
  limits.virtual-machines=4 \
  limits.cpu=30 \
  limits.memory=48GiB \
  limits.processes=16384 \
  limits.disk=700GiB
sudo incus project unset "$project" restricted.devices.disk.paths

if ! sudo incus network acl show tcompute-egress --project default >/dev/null 2>&1; then
  sudo incus network acl create tcompute-egress --project default
fi

sudo incus network acl edit tcompute-egress --project default <<'"'"'EOF'"'"'
config: {}
description: Tashan Compute public egress with host, private, metadata and reserved ranges denied
egress:
- action: reject
  destination: 0.0.0.0/8,10.0.0.0/8,100.64.0.0/10,127.0.0.0/8,169.254.0.0/16,172.16.0.0/12,192.0.0.0/24,192.168.0.0/16,198.18.0.0/15,224.0.0.0/4
  description: Deny IPv4 host, private, metadata, multicast and selected reserved destinations
  state: enabled
- action: reject
  destination: ::/128,::1/128,fc00::/7,fe80::/10,ff00::/8
  description: Deny IPv6 host, private, link-local and multicast destinations
  state: enabled
- action: allow
  destination: 0.0.0.0/0,::/0
  description: Allow public Internet egress
  state: enabled
ingress: []
EOF

sudo incus network set "$network" --project default \
  security.acls=tcompute-egress \
  security.acls.default.ingress.action=allow \
  security.acls.default.egress.action=allow

sudo incus project show "$project"
sudo incus network show "$network" --project default
sudo incus network acl show tcompute-egress --project default
'

echo 'Incus confinement configured and verified'
