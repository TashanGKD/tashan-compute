# Security Policy

## Public source is not platform access

Tashan Compute is distributed from a public repository so users can install and inspect its Skill and CLI. Cloning, modifying, building or installing this repository does not create an account, grant an organization membership, issue a trusted token, expose AUP SSH, or grant administrator privileges.

The server is authoritative for identity and authorization. Every protected request is checked against a server-signed session, the current device state, the current account state, the requested space and the role stored in the production database. Client-provided role claims are not trusted.

## Secrets policy

Do not commit or report real passwords, initial passwords, access or refresh tokens, signing keys, database URLs, object-storage credentials, SSH details, deployment keys or production environment files.

The repository must contain only redacted examples and variable names. Local `.env` files, private keys, credentials, runtime data and generated tokens are ignored by Git.

## Reporting a vulnerability

Do not open a public issue containing credentials, private user data or an exploitable production proof. Report the affected component, expected boundary and a minimally redacted reproduction privately to the repository maintainers. Rotate any credential that may have been exposed before continuing investigation.
