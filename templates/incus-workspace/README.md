# Tashan Compute Standard Workspace

This template creates an unprivileged Ubuntu 24.04 Incus compute container and
a separate persistent `/home/coder` volume. Stopping destroys the 8 GiB compute
container and retains the home volume; starting recreates compute with a fresh
Coder token. Personal homes are 50 GiB. A 500 GiB organization home is accepted
only when the workspace owner is the platform account `tashan-admin`.

The image installs Python, Node.js, Go, Rust, C/C++, PostgreSQL, Redis and common
development tools plus Podman/Buildah exposed through a Docker-compatible
`docker build` command. The workspace allows nested namespaces but remains
unprivileged and receives no host Docker socket, Incus socket or host path.
