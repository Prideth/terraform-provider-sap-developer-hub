# Security Policy

## Reporting a vulnerability

If you find a security issue in this provider (for example, a way
credentials could leak into logs, state, or diagnostics, or a request
forgery / injection issue in the API client), please open a private
[GitHub Security Advisory](https://github.com/Prideth/terraform-provider-sap-developer-hub/security/advisories/new)
rather than a public issue.

## Scope

This provider only talks to the Developer Hub API host you configure it
with, using the OAuth2 credentials you supply (see `DESIGN.md` §4/§16).
It does not itself provision or manage SAP BTP account/authorization
resources - see `README.md#scope`.

## Secret handling

- `client_secret` (provider configuration) and any Developer Hub secret
  this provider reads are marked `Sensitive` in the Terraform schema, are
  never logged, and are never included in a diagnostic message.
- Access tokens are held only in memory (`internal/client/auth`) and are
  never written to Terraform state or logs.
- See `DESIGN.md` §16 for the full accounting of what is, and is not,
  stored in state.

## Supported versions

Only the latest tagged release is supported with security fixes while this
provider is pre-1.0.
