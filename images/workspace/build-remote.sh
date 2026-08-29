#!/usr/bin/env bash
set -euo pipefail

alias_name=${1:-}
base_fingerprint=${2:-}
coder_version=${3:-}
packages_file=${4:-}
[[ "$alias_name" =~ ^tcompute-workspace-[a-z0-9-]{1,40}$ ]]
[[ "$base_fingerprint" =~ ^[a-f0-9]{64}$ ]]
[[ "$coder_version" =~ ^[0-9.]{5,16}$ ]]
[[ "$packages_file" == /home/aup/tashan-compute/staging/workspace-packages.txt ]]

builder=tcompute-workspace-image-builder
cleanup() {
  sudo -u tcompute -H incus delete -f "$builder" >/dev/null 2>&1 || true
}
trap cleanup EXIT INT TERM
if sudo -u tcompute -H incus image alias show "$alias_name" >/dev/null 2>&1; then
  echo 'refusing: image alias already exists' >&2
  exit 1
fi
cleanup
sudo -u tcompute -H incus launch "$base_fingerprint" "$builder" -c security.nesting=true
for attempt in $(seq 1 30); do
  sudo -u tcompute -H incus exec "$builder" -- true 2>/dev/null && break
  sleep 1
done
sudo -u tcompute -H incus file push "$packages_file" "$builder/tmp/workspace-packages.txt"
sudo -u tcompute -H incus exec "$builder" -- sh -euc '
  apt-get update
  DEBIAN_FRONTEND=noninteractive xargs -a /tmp/workspace-packages.txt apt-get install -y
'
sudo -u tcompute -H incus exec "$builder" -- sh -euc '
  expected_version=$1
  id coder >/dev/null 2>&1 || useradd -m -s /bin/bash coder
  usermod -aG sudo coder
  printf "%s\n" "coder ALL=(ALL) NOPASSWD:ALL" >/etc/sudoers.d/coder
  chmod 0440 /etc/sudoers.d/coder
  curl -fsSL --retry 10 --retry-delay 5 https://compute.tashan.chat/bin/coder-linux-amd64 -o /usr/local/bin/coder
  chmod 0755 /usr/local/bin/coder
  /usr/local/bin/coder version | grep -F "Coder v$expected_version"
  ln -sfn /usr/bin/podman /usr/local/bin/docker
  printf "%s\n" "#!/bin/sh" "set -eu" "test ! -S /var/lib/incus/unix.socket" "test ! -S /var/run/docker.sock" "test \"\$(id -u)\" = 0" "echo workspace-boundary=PASS" >/usr/local/sbin/tcompute-boundary-check
  chmod 0755 /usr/local/sbin/tcompute-boundary-check
  cloud-init clean --logs --machine-id
  apt-get clean
' _ "$coder_version"
sudo -u tcompute -H incus stop "$builder"
sudo -u tcompute -H incus publish "$builder" --alias "$alias_name"
sudo -u tcompute -H incus image show "$alias_name" | sed -n '1,35p'
cleanup
trap - EXIT INT TERM
echo workspace-image-build=PASS
