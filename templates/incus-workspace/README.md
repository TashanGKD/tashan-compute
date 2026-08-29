# Tashan Compute Standard Workspace

This Coder template creates one persistent, unprivileged Ubuntu 24.04 Incus
container in the dedicated `user-955` project. Stopping a workspace stops the
container without deleting its 50 GiB root filesystem; deleting the workspace
deletes the container.

The image installs Python, Node.js, Go, Rust, C/C++, PostgreSQL, Redis and common
development tools. The `coder` user has passwordless sudo only inside the
container. The workspace has no host Docker socket, Incus socket, host path or
privileged/nested-container mode. A separate template is required for rootless
container-image builds.
