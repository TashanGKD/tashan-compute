# Authentication

- There is no self-registration. A platform administrator creates an email, username and temporary password.
- `tcompute login --email <email>` reads the password from a hidden prompt and stores only the Coder session token in macOS Keychain or Linux Secret Service.
- On a temporary-password handoff, the first action must be `tcompute password change`; after success, log in again. Do not perform workspace operations first.
- `tcompute admin user reset-password <username>` reads the new password from protected input. Coder revokes all old device sessions when the reset succeeds.
- One person has one account. Multiple computers create multiple device sessions; do not create device-shaped subaccounts.
- Installing the public repository, Skill or CLI grants no account, organization membership, server shell or administrator role.
