# Authentication

- Platform administrators create usernames and initial passwords; there is no self-registration.
- `tcompute auth login` uses a hidden password prompt and stores the resulting device session in the operating-system credential store.
- An initial-password session can only change its password or log out. After the change, every existing session is revoked and the user logs in again.
- An administrator password reset revokes every old device session for that account.
- Installing or modifying the public CLI cannot create a trusted Token. Server signatures, current device state and database roles are authoritative.
