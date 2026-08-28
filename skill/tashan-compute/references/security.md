# Safety boundaries

The public repository contains no account, password, Token, signing key, database credential, object-storage credential, deployment key or AUP SSH access.

Require explicit confirmation for permanent deletion, overwrite, service publication, database restore, Secret grants and administrator mutations. `service public` means anonymous internet users can access the service.

Reject any workflow that attempts to bypass the API through SSH, Tailscale, a host shell, Docker socket, host port mapping or direct object-storage credentials. A missing capability is an implementation gap, not permission to bypass the platform boundary.
