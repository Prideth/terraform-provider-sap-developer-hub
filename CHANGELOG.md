# Changelog

## Unreleased

### Added

- Initial provider foundation: OAuth2 client-credentials authentication,
  retrying HTTP client with CSRF support, and the Developer Hub domain
  client, built against endpoints independently verified from SAP's own
  documentation (see `DESIGN.md`).
- `developerhub_application`, managing Developer Hub applications and their
  custom attributes.
- `developerhub_product_subscription`, managing an application's product
  subscriptions.
- `developerhub_current_user` and `developerhub_registered_users` data
  sources, so a `developer_id` never needs to be copied from the Developer
  Hub UI by hand.
- `developerhub_applications` and `developerhub_product_subscriptions` data
  sources, listing applications and subscriptions (including ones created
  in the Developer Hub UI) with optional filters.
- Server-side pagination (`__next`) for every collection read, refusing
  paging links that point to a different host.
- `make verify-api`: checks the provider against the live tenant's OData
  `$metadata` and `/api/1.0/` endpoints.
- `DESIGN.md`, documenting the full scope boundary with
  `terraform-provider-integration-suite` and the SAP BTP provider, the API
  research behind every implemented (and deliberately not implemented)
  feature, and the provider's error handling, retry and secret-handling
  conventions.
