# Design

This document records the scope, architecture, and API research behind the
Terraform provider for the SAP Integration Suite Developer Hub. It is the
reference for every design decision in this repository, and it is kept up to
date as the provider grows.

## 1. Scope

This provider manages **Developer Hub catalog and consumer-lifecycle
configuration** for a Developer Hub that already exists and is already
reachable:

- Developer Hub applications, their custom attributes, and their product
  subscriptions.
- Developer Hub developer/registration identity data (read-only).

It assumes the following are already true of the target tenant, and does not
attempt to create or verify them beyond a clear error message:

- The SAP Integration Suite subscription exists.
- The API Management capability, and the Developer Hub sub-capability, have
  been activated.
- A `devportal-apiaccess` service instance and service key exist for
  programmatic access.

## 2. Non-Scope

The following are explicitly **not** managed by this provider. They belong
either to SAP BTP control-plane tooling (the official `SAP/btp` Terraform
provider, or Cloud Foundry/CLI tooling) or to
[`Prideth/terraform-provider-integration-suite`](https://github.com/Prideth/terraform-provider-integration-suite),
which this provider is designed to be used alongside, never in competition
with:

| Out of scope here | Where it belongs |
|---|---|
| BTP global accounts, directories, subaccounts | `SAP/btp` provider |
| Cloud Foundry orgs, spaces, service instances/keys | `SAP/btp` / `cloudfoundry` providers |
| Entitlements, service plan assignment | `SAP/btp` provider |
| SAP Integration Suite subscription itself | `SAP/btp` provider |
| Activating the API Management / Developer Hub **capability** | `terraform-provider-integration-suite` |
| BTP role collections, general BTP user/IdP management | `SAP/btp` provider |
| API Proxies, API Products (`APIProducts`), API Providers, Key/Value Maps, Certificates | `terraform-provider-integration-suite` (`internal/client/apimanagementclassic`) |
| Integration flows, packages, value mappings, security material | `terraform-provider-integration-suite` |

A product is authored and published from **within SAP Integration Suite /
API Portal** (`create-a-product-d769622.md`), using the classic API
Management `APIProducts` OData service
(`/apiportal/api/1.0/Management.svc/APIProducts`). That service, and the
resource that wraps it, already exists in
`terraform-provider-integration-suite` (`apimanagementclassic/api_product.go`
→ presumably `sapintegrationsuite_api_product` or equivalent). This provider
therefore does **not** re-implement product/API-bundle authoring — doing so
would create two providers able to write the same object, which directly
contradicts the stated boundary. Where this provider's resources need to
reference a product (for a subscription), they take the product's technical
name as a plain string argument, the same identifier the Integration Suite
provider's product resource already exposes.

If the tenant does not have Developer Hub reachable, every resource and data
source in this provider fails with a message of the shape:

```
Developer Hub is not reachable for the configured tenant.
Enable the Developer Hub capability through the SAP Integration Suite
provider first, then create a "devportal-apiaccess" service instance
and provide its service key to this provider.
```

## 3. Developer Hub Architecture (as documented by SAP)

Source: `help.sap.com/docs/integration-suite` (mirrored on GitHub at
[`SAP-docs/btp-integration-suite`](https://github.com/SAP-docs/btp-integration-suite),
used in this repository's research because the live help.sap.com/api.sap.com
sites were not reachable from the development sandbox's network policy — see
§13).

- Developer Hub is "a web-based platform that serves as a centralized
  catalog for APIs, events, and MCP servers"
  (`developer-hub-41f7c45.md`). It is one of the sub-capabilities of API
  Management within SAP Integration Suite.
- Content (APIs, Events, MCP servers) is bundled into **Products** inside
  SAP Integration Suite / API Portal, then published to the Developer Hub
  catalog.
- Developer Hub can also run as a **Centralized Developer Hub**, accepting
  published content from up to three SAP Integration Suite API Management
  tenants (`centralized-developer-hub-38422de.md`,
  `create-a-connection-request-for-the-centralized-developer-hub-c7bda8c.md`).
- Application developers register (self-service or admin-provisioned),
  create **Applications**, and **Subscribe** those applications to
  Products in order to consume the bundled APIs/Events/MCP servers.

## 4. Authentication

Source: `accessing-developer-hub-apis-programmatically-dabee6e.md`.

Developer Hub exposes a programmatic API surface behind a dedicated Cloud
Foundry service plan: **`devportal-apiaccess`**, on the **API Management,
Developer Hub** service in the Service Marketplace. A service instance is
created with one of four role-collection payloads:

| Role | Payload | Grants |
|---|---|---|
| `AuthGroup.API.Admin` | `{"role": "AuthGroup.API.Admin"}` | Full admin access to applications, attributes, API packages, APIs and products, app developer and metering endpoints. |
| `AuthGroup.Content.Admin` | `{"role": "AuthGroup.Content.Admin"}` | Manage domain categories and product-to-category assignment. |
| `AuthGroup.API.ApplicationDeveloper` | `{"role": "AuthGroup.API.ApplicationDeveloper", "developerId": "<id>"}` | Scoped access to a single developer's applications, API packages, and product/API proxy read access. `developerId` must be a *registered* Developer Hub developer id (see §7); registering a new developer through this API is documented as unsupported (self-service onboarding cannot be automated through the API — see §11). |
| `AuthGroup.External.Reviewer` | `{"role": "AuthGroup.External.Reviewer"}` | Used by an external approval workflow (e.g. SAP Build Process Automation) to approve/reject subscription requests under External Governance. |

A service key (of type `binding-secret` or `x509`) yields:

```json
{
  "url": "https://<devportal-application-url>",
  "tokenUrl": "https://<token-endpoint>/oauth/token",
  "clientId": "...",
  "clientSecret": "..."
}
```

The provider authenticates with the **OAuth2 client-credentials grant**
against `tokenUrl` (`Authorization: Basic base64(clientId:clientSecret)`,
`grant_type=client_credentials`), then calls the Developer Hub API at `url`
with `Authorization: Bearer <token>`. x509-based service keys are out of
scope for the first release (documented as a `Known Gap`, see §12) because
they require client-certificate transport configuration that has no
precedent in the reference provider and needs its own design pass.

Provider configuration mirrors `terraform-provider-integration-suite`'s
pattern: Terraform attributes with environment-variable fallbacks.

| Attribute | Environment variable | Required | Sensitive |
|---|---|---|---|
| `url` | `SAP_DEVELOPER_HUB_URL` | yes | no |
| `token_url` | `SAP_DEVELOPER_HUB_TOKEN_URL` | yes | no |
| `client_id` | `SAP_DEVELOPER_HUB_CLIENT_ID` | yes | no |
| `client_secret` | `SAP_DEVELOPER_HUB_CLIENT_SECRET` | yes | **yes** |

## 5. SAP API Inventory

This is the API matrix required by the project brief. "Verified" means the
concrete request/response shape is quoted or shown in an official SAP
document read during this project (all under
`docs/ISuite_Integrations_APIs/` in the `SAP-docs/btp-integration-suite`
mirror). "Documented to exist, schema unverified" means SAP's documentation
confirms a public, supported API exists (service plan, role collection, or
an `api.sap.com` artifact link) but the concrete wire contract was not
independently confirmed, because both `help.sap.com` and `api.sap.com` (the
canonical source for the OpenAPI/Swagger specs themselves) were blocked by
the outbound network policy of the sandbox this provider was built in (see
§13). No resource is implemented against an unverified schema — see §12.

| Feature | SAP doc | Public API | Verified? | Endpoint (as documented) | Auth | Terraform resource | Terraform data source | Import | Status |
|---|---|---|---|---|---|---|---|---|---|
| Applications | `add-custom-attributes-to-an-application-39c3cbd.md`, `custom-attributes-90a5a6d.md` | Yes | **Verified** (sample payloads) | `POST /odata/1.0/data.svc/APIMgmt.Applications` on the Dev-Portal host | OAuth2 client-credentials (devportal-apiaccess) | `developerhub_application` | – | by application id | Implemented |
| Application custom attributes | `custom-attributes-90a5a6d.md` | Yes | **Verified** | `POST .../APIMgmt.Applications(<id>)/ToAttributes`, `PUT`/`DELETE .../APIMgmt.Attributes(name=..,entityId=..,entityType='Applications')` | same | modeled as a nested `attribute` block on `developerhub_application` | – | via parent application | Implemented |
| Product subscriptions | `custom-attributes-90a5a6d.md` (nested `ToSubscriptions`/`ToAPIProduct` in the Applications create payload), `manage-product-subscriptions-452fcef.md` | Yes | **Partially verified**: the nested create shape is shown verbatim; standalone read/update/delete of a single subscription entity were not shown and are implemented following the same OData conventions used elsewhere in this API family (see §9) | `.../APIMgmt.Applications(<id>)/ToSubscriptions` | same | `developerhub_product_subscription` | – | `<application_id>/<product_name>` | Implemented, with a documented inference (§9) |
| Developer / current user identity | `accessing-developer-hub-apis-programmatically-dabee6e.md` | Yes | **Verified** (literal example request/response) | `GET /api/1.0/user` | Bearer token | – | `developerhub_current_user` | n/a (read-only) | Implemented |
| Registered developers | `accessing-developer-hub-apis-programmatically-dabee6e.md` | Yes | **Verified** (literal example response) | `GET /api/1.0/registrations?type=registered` | Bearer token | – | `developerhub_registered_users` | n/a | Implemented |
| Bulk user registration (admin) | `registering-on-developer-hub-c85fafe.md` | Yes, named artifact | Documented to exist, schema unverified: `api.sap.com/api/DevPortal_RegisteringUsers_CF/resource` | unknown exact path/payload | OAuth2 | not implemented | not implemented | – | **SAP API unavailable to verify** |
| Domain categories | `manage-domain-categories-bd9691d.md` | Not found | No REST endpoint documented; UI-only procedure (Admin Center → Manage Domain Categories) | – | – | not implemented | not implemented | – | **No documented public API** |
| Access governance (site visibility: Authorized/Authenticated/All Visitors) | `manage-access-ad1b441.md` | Not found | UI-only procedure. Confirmed: up to 5 minutes of eventual consistency after a change. | – | – | not implemented | not implemented | – | **No documented public API** |
| Subscription governance (Auto Approval / Manage Approval Outside Developer Hub) | `configure-governance-settings-for-apis-and-events-7446560.md`, `manage-product-subscriptions-452fcef.md` | Not found | UI-only procedure; the two modes are confirmed by name, no confirmed enum wire value | – | – | not implemented | not implemented | – | **No documented public API** |
| External governance (SPI destination) | `configure-governance-settings-for-apis-and-events-7446560.md` | Partial | The external side is a **destination the customer configures in BTP Cockpit** pointing at their own SPI implementation — this is BTP Cockpit config, not a Developer Hub write API | – | – | out of scope (destination = BTP concern) | not implemented | – | **Out of scope / no public write API** |
| MCP Server products | `discover-and-publish-mcp-servers-from-sap-integration-suite-9db32d0.md` | Not found as a distinct API | UI-only procedure ("Create Product" dialog on the MCP Servers tab); no separate API beyond the product-authoring API that already belongs to the other provider | – | – | not implemented (belongs to product authoring, out of scope, §2) | not implemented | – | **Out of scope / no documented public API** |
| Business systems / content discovery | `manage-content-4b89a8b.md` | Not found | UI-only ("Manage Content" lists registered business systems); registration itself is a BTP System Landscape action | – | – | not implemented | not implemented | – | **No documented public API found; likely belongs to BTP system landscape tooling** |
| Site configuration / branding | `customize-the-visual-format-of-developer-hub-2eacd52.md` | Not found | UI-only (Site Editor) | – | – | not implemented | not implemented | – | **No documented public API** |
| Notifications | `manage-notifications-df32457.md` | Not found | UI-only (Admin Center → Notifications) | – | – | not implemented | not implemented | – | **No documented public API** |
| Centralized Developer Hub connections | `create-a-connection-request-for-the-centralized-developer-hub-c7bda8c.md` | Not found for the Developer Hub side | UI-only from the Developer Hub admin side (Manage Connections). The API Management tenant side generates OAuth credentials for the `APIPortal.Service.CatalogIntegration` role, but no Developer Hub write API to submit/approve a connection request was found. **Explicitly documented as irreversible**: "the option to disconnect ... isn't supported currently", and re-connecting to a different centralized Developer Hub is also not supported once established. | – | – | not implemented | not implemented | – | **No documented public API; and see §10 on irreversibility** |
| Product visibility / discovery & subscription permissions (custom role collections) | `configuring-permissions-for-products-e4636bf.md` | Not found | UI-only; the actual permission grant is a **BTP custom role collection**, assigned in BTP Cockpit, out of this provider's scope by definition (§2) | – | – | out of scope | not implemented | – | **Out of scope (BTP role collections)** |
| Applications: admin-created on behalf of a developer, app key/secret handover | `add-custom-attributes-to-an-application-39c3cbd.md` | Same API as Applications | Verified (same `APIMgmt.Applications` entity, `developer_id` field) | same as Applications | same | supported via `developer_id` argument on `developerhub_application` | – | – | Implemented |

Package/API/Event/MCP-server discovery *within* a business system, and
Product/domain-category/governance/site/notification/connection management,
are the clearest candidates for a **Phase 2** of this provider once the
corresponding public API contracts can be verified against `api.sap.com`
(which requires network access this sandbox did not have — see §13). They
are listed in `FEATURE_MATRIX.md` with status `needs-api-verification`, not
silently dropped.

## 6. Resource Mapping

| Terraform type | Backing API | Lifecycle |
|---|---|---|
| `developerhub_application` | `APIMgmt.Applications` (OData, devportal) | Full CRUD on `id` (Computed, server-assigned), `title`, `version`, `developer_id`, and nested `attribute` blocks (`ToAttributes`). SAP's UI documentation (`add-custom-attributes-to-an-application-39c3cbd.md`) confirms every application also has a generated app key/secret ("handover the application key and secret to that user"), but no JSON sample shows their actual field names, so this provider does **not** guess them — `app_key`/`app_secret` are tracked as a documented gap (§12) rather than implemented under an invented field name. |
| `developerhub_product_subscription` | `APIMgmt.Applications(<id>)/ToSubscriptions` | Create/Read/Delete. The sample create payload shows each subscription carrying its own server-assigned `id`, distinct from the product it targets (`ToAPIProduct`), so the subscription is its own keyed child entity of the application rather than being addressable by product name alone. No update endpoint is documented; changing which product a subscription targets is a replace (`RequiresReplace` on `product_name`). |

## 7. Data Source Mapping

| Terraform type | Backing API | Notes |
|---|---|---|
| `developerhub_current_user` | `GET /api/1.0/user` | Returns the identity of the credentials used to authenticate — useful to discover your own `developer_id` for `developerhub_application.developer_id`, exactly as the SAP doc describes obtaining it (§4/§11). |
| `developerhub_registered_users` | `GET /api/1.0/registrations?type=registered` | Lists registered Developer Hub developers and their `userId` (== `developer_id`), so a Product Subscription/Application configuration never needs an internal ID copied by hand from the UI. |

## 8. Terraform IDs

- `developerhub_application`: the Terraform ID is the SAP-assigned
  application id (the `id` field of `APIMgmt.Applications`, a GUID-shaped
  string). Import: `terraform import developerhub_application.example <id>`.
- `developerhub_product_subscription`: composite ID
  `<application_id>/<subscription_id>`. The sample create payload
  (`custom-attributes-90a5a6d.md`) shows a subscription as its own keyed
  child entity of an application (its own `id`, plus a `ToAPIProduct`
  navigation to `APIMgmt.APIProducts('<ProdName>')`), server-assigned on
  create the same way an application's own `id` is. Import:
  `terraform import developerhub_product_subscription.example <application_id>/<subscription_id>`.

Display names/titles are never used as Terraform IDs; SAP's own payloads use
stable technical identifiers (`id` for applications, the product's technical
`name` for the product reference), and this provider follows that.

## 9. Documented Inference: Subscriptions

SAP's documentation shows the **nested create** shape for a subscription
(embedded inside an `APIMgmt.Applications` create payload, or via
`POST .../APIMgmt.Applications(<id>)/ToSubscriptions`) but does not show a
worked example of reading, updating, or deleting a single subscription by
itself. Every other entity in this same API family that SAP *does* fully
document (`APIMgmt.Attributes`) follows plain OData-style per-entity
addressing: `GET`/`PUT`/`DELETE Entity(key1=..,key2=..)`, with
`x-csrf-token: fetch` required ahead of any write.
`developerhub_product_subscription` follows that same, already-proven
convention: it addresses a single subscription as
`APIMgmt.Applications('<app_id>')/ToSubscriptions('<subscription_id>')` for
`GET`/`DELETE`, using the `id` SAP assigns and returns on create. This is
called out explicitly, both here and in the resource's own doc comment, as
a **documented inference** rather than a verified fact — if SAP's actual
addressing differs, the resource's `Read` will surface that as a normal API
error rather than crash, and the gap is tracked in `FEATURE_MATRIX.md`.

## 10. Centralized Developer Hub

Connecting an SAP Integration Suite API Management tenant to a centralized
Developer Hub is explicitly documented as **irreversible**:

> "The option to disconnect an SAP Integration Suite API Management tenant
> from an existing Developer Hub isn't supported currently."
>
> "Once this connection is set up, you can't place a request to sever this
> connection and establish a new connection with any other centralized
> Developer Hub."

No Developer Hub-side write API for connection requests was found (§5), so
this provider implements nothing here. If such an API is found in a future
verification pass, any resource built on it must never be given
`ForceNew`/replace semantics implicitly (Terraform must never recreate an
irreversible connection just because a dependent attribute changed), and the
plan output must make the irreversibility of `create` and `delete` visible
to the operator before `apply`.

## 11. Eventual Consistency

SAP explicitly documents that Access Governance changes ("Manage Access")
can take **up to 5 minutes** to propagate
(`manage-access-ad1b441.md`). No resource for Access Governance is
implemented yet (§5), but the provider's shared HTTP client already supports
the retry/backoff primitives such a resource would need
(`internal/client/http`), and any future resource here must implement
read-after-write polling with a bounded, configurable timeout rather than
trusting an immediate `Read` after `Create`/`Update`.

## 12. Known SAP API Gaps

These are documented gaps, not invented workarounds:

- **Domain categories** — no public REST API found. Not implemented.
- **Access governance / subscription governance settings** — no public REST
  API found for reading or writing the setting itself (only for what it
  governs). Not implemented.
- **Site configuration / branding** — no public REST API found. Not
  implemented.
- **Notifications** — no public REST API found. Not implemented.
- **Centralized Developer Hub connections** — no public REST API found on
  the Developer Hub side, and the action is explicitly irreversible even in
  the UI. Not implemented.
- **Bulk user registration** (`DevPortal_RegisteringUsers_CF`) — a real,
  named `api.sap.com` artifact, but its schema could not be verified in this
  sandbox (network policy blocked `api.sap.com`). Not implemented; tracked
  for a follow-up verification pass.
- **x509-based service keys** — the `devportal-apiaccess` plan supports
  certificate-based service keys in addition to `binding-secret`. Only
  `binding-secret` (client id/secret) is implemented in the first release.
- **Application `app_key`/`app_secret`** — confirmed by name in SAP's UI
  documentation, but no JSON sample shows the actual field names SAP's API
  uses for them, so this provider does not expose them. See §6.

None of these are stubbed with a guessed endpoint. Each is either absent
from the provider entirely, or (for governed *content*, like MCP-server
products and business-system discovery) explicitly deferred to
`terraform-provider-integration-suite` per the scope boundary in §2.

## 13. Research Constraint: Network Policy

The development sandbox this provider was built in blocks outbound HTTPS to
`help.sap.com`, `api.sap.com`, `community.sap.com`, and `developers.sap.com`
at the egress proxy level (`connect_rejected`, organization policy). This
made it impossible to read the SAP Help Portal or SAP Business Accelerator
Hub sites directly, which is where the project brief asks for verification
to happen first.

The research in this document instead comes from
[`SAP-docs/btp-integration-suite`](https://github.com/SAP-docs/btp-integration-suite),
SAP's own GitHub mirror of the same Help Portal Markdown source content,
which was reachable. It is the same content, at the same fidelity, as
`help.sap.com/docs/integration-suite` — but it does not include the
`api.sap.com` OpenAPI/Swagger specifications themselves, only the Help
Portal's prose, procedures, and the sample request/response payloads authors
chose to inline (which is where the verified rows in §5 come from). Anyone
continuing this project with normal network access to `api.sap.com` should
treat completing §5/§12 as the highest-value next step.

## 14. Error Handling

The client stack maps every SAP error response to a single normalized
`apierror.Error` (status code, SAP error code, message, details), following
`terraform-provider-integration-suite`'s pattern. Resource/data source
`Read`/`Create`/`Update`/`Delete` methods surface it as:

```
Unable to <verb> Developer Hub application "<name>":
SAP API returned HTTP 409, code "...":
<message>
```

Never the raw HTML/JSON body, never a token or header value.

## 15. Retry Strategy

Copied near-verbatim from `terraform-provider-integration-suite`'s
`internal/client/http`: exponential backoff with full jitter on `429`,
`502`, `503`, `504`; `Retry-After` header honored when present; a single
retry after invalidating the cached token on `401`; CSRF token fetch-and
retry-once on a `403` carrying `X-CSRF-Token: Required`, which this API
family requires for write operations (seen in the `Attributes` sample
payloads). This logic is intentionally protocol-agnostic so it did not need
to change to fit Developer Hub's OData v1.0 service instead of Integration
Suite's OData v2 services.

## 16. Secret Handling

- `client_secret` (provider config): `Sensitive`, sourced from Terraform
  config or `SAP_DEVELOPER_HUB_CLIENT_SECRET`, never logged.
- `app_secret` (on `developerhub_application`): `Sensitive, Computed`.
  SAP generates and returns it once, on create; it is stored in state (as
  Terraform requires for any value a practitioner may need downstream, e.g.
  to wire into another system) but never printed in a diagnostic, never
  reread from the API on `Read` (there is nothing to reread — SAP does not
  echo it back), and a plan after `apply` shows no diff for it because it is
  `Computed` with `UseStateForUnknown`.
- Access tokens: held only in memory by `internal/client/auth`, never
  written to state or logs.

## 17. Import Strategy

See §8. Both implemented resources support `terraform import`.

## 18. Follow-Up Research Pass

A second research pass specifically targeted business system content
discovery and MCP server/API artifact invocation, since those looked most
likely to hide an additional public API:

- `discover-and-publish-apis-from-business-systems-0cea56f.md`: business
  system registration is entirely an SAP BTP System Landscape / Global
  Account Administrator action (outside this provider's scope by
  definition, §2), and fetching APIs/events from a registered business
  system inside Developer Hub is documented only as a UI procedure ("Manage
  Content" → "Business Systems" tab). No REST endpoint found.
- `invoke-an-api-artifact-by-obtaining-credentials-via-developer-hub-e79810f.md`
  and the equivalent MCP server page: confirm that a subscription's
  application has a Key/Secret/Token URL visible on its "Overview" tab in
  the UI - i.e., further confirmation that `developerhub_application` has
  generated credentials (consistent with §12's `app_key`/`app_secret` gap)
  - but again as a UI screen, not a documented JSON API response, so this
  still does not add a verifiable field name to implement against.
  `configure-mcp-server-access-using-default-authentication-xsuaa-dc283ab.md`:
  MCP server "Agent subscriptions" also generate a client ID/secret through
  Developer Hub, but again only described as a UI flow ("Developer Hub
  automatically generates the client ID and client secret").

None of this changes §5's matrix or §12's gap list - it corroborates them.

## 19. Future Extensions

In priority order, contingent on verifying the underlying API against
`api.sap.com`:

1. Domain categories (content organization) — highest practitioner value,
   per the project brief.
2. Access governance and subscription governance settings.
3. Bulk/admin user registration (`DevPortal_RegisteringUsers_CF`).
4. Site configuration/branding, if a public API surfaces for it.
5. Centralized Developer Hub connections, if a public write API surfaces —
   with mandatory irreversibility warnings in the resource docs and no
   forced recreation.
