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
| Applications | `add-custom-attributes-to-an-application-39c3cbd.md`, `custom-attributes-90a5a6d.md`, `create-or-update-or-read-an-application-using-subscription-key-e2645b5.md` | Yes | **Verified** (multiple worked create/update/read/delete examples, including exact `app_key`/`app_secret` field names) | `POST/GET/PUT/DELETE /odata/1.0/data.svc/APIMgmt.Applications(<id>)` on the Dev-Portal host | OAuth2 client-credentials (devportal-apiaccess) | `developerhub_application` | – | by application id | Implemented |
| Application custom attributes | `custom-attributes-90a5a6d.md`, `api-specs/DevPortal_Application_CF.yaml` | Yes | **Verified against the specification** (2026-09-26). The spec differs from the older Help Portal sample in three places, and the provider follows the spec: the `entityType` key is `'applications'` (lowercase), an update sends only `{"value": ...}`, and the attribute type has no `entityType` property. Name at most 235 characters, value 1024. | `POST .../APIMgmt.Applications('{id}')/ToAttributes`, `GET`/`PUT`/`DELETE .../APIMgmt.Attributes(name=..,entityId=..,entityType='applications')` | same | modeled as a nested `attribute` block on `developerhub_application` | – | via parent application | Implemented |
| Product subscriptions | `create-or-update-or-read-an-application-using-subscription-key-e2645b5.md` (full EDMX + worked create/update payloads for the top-level `APIMgmt.Subscriptions` entity), `manage-product-subscriptions-452fcef.md` | Yes | **Verified**: create (`POST`) and update (`PUT`) shown verbatim against `APIMgmt.Subscriptions(<id>)` directly; delete is a high-confidence inference by analogy with the sibling `APIMgmt.Applications` entity's documented `DELETE` (see §9) | `POST/GET/PUT/DELETE /odata/1.0/data.svc/APIMgmt.Subscriptions(<id>)` | same | `developerhub_product_subscription` | – | by subscription id | Implemented |
| Developer / current user identity | `accessing-developer-hub-apis-programmatically-dabee6e.md` | Yes | **Verified** (literal example request/response) | `GET /api/1.0/user` | Bearer token | – | `developerhub_current_user` | n/a (read-only) | Implemented |
| Registered developers | `accessing-developer-hub-apis-programmatically-dabee6e.md` | Yes | **Verified** (literal example response) | `GET /api/1.0/registrations?type=registered` | Bearer token | – | `developerhub_registered_users` | n/a | Implemented |
| Developer registration ("Add User", accept/reject/revoke) | `registering-on-developer-hub-c85fafe.md`, `managing-the-access-request-of-the-users-8b79ee8.md`, `api-specs/DevPortal_RegisteringUsers_CF.yaml` | Yes | **Verified against the specification** | `POST /api/1.0/registrations` (an administrator registers the user directly), `GET /api/1.0/registrations?type=registered\|pending`, `PUT /api/1.0/registrations/{developerid}` (status `registered`, `rejected` or `revoked`) | OAuth2, admin service key | `developerhub_developer` | `developerhub_registration_requests` (pending) | by user id | Implemented |
| Product catalog | `api-specs/DevPortal_Application_CF.yaml` (`developer.APIProductsType`, `APIProducts` entity set), SAP blog listing `APIMgmt.APIProducts` | Yes (read) | **Verified against the specification** | `GET /odata/1.0/data.svc/APIMgmt.APIProducts` | same | – (authoring stays in Integration Suite, §2) | `developerhub_products` | n/a | Implemented |
| Domain categories | `manage-domain-categories-bd9691d.md` | Not found | No REST endpoint documented; UI-only procedure (Admin Center → Manage Domain Categories) | – | – | not implemented | not implemented | – | **No documented public API** |
| Access governance (site visibility: Authorized/Authenticated/All Visitors) | `manage-access-ad1b441.md` | Not found | UI-only procedure. Confirmed: up to 5 minutes of eventual consistency after a change. | – | – | not implemented | not implemented | – | **No documented public API** |
| Subscription governance (Auto Approval / Manage Approval Outside Developer Hub) | `configure-governance-settings-for-apis-and-events-7446560.md`, `manage-product-subscriptions-452fcef.md` | Not found | UI-only procedure; the two modes are confirmed by name, no confirmed enum wire value | – | – | not implemented | not implemented | – | **No documented public API** |
| External governance (SPI destination, decision callbacks) | `configure-governance-settings-for-apis-and-events-7446560.md`, `setting-up-external-governance-992512e.md`, SAP's "Part 3: Implementing External Governance in Developer Hub" (community.sap.com, 2025-12-17) | Yes, but not declarative | The external side is a **BTP destination** (`DeveloperHub_Governance_SPI`) pointing at the customer's own SPI implementation — BTP configuration, out of scope. `DevPortal_ExternalGovernance_CF` has exactly two resources, `Governance_Decision` (approve/reject one pending subscription request) and `Deletion_Confirmation` (confirm credential invalidation): one-off workflow callbacks with no desired state for Terraform to converge on, so they are deliberately not modeled as resources. | – | `AuthGroup.External.Reviewer` | deliberately not a resource | not implemented | – | **Not a Terraform fit / out of scope** |
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
| `developerhub_application` | `APIMgmt.Applications` (OData, devportal) | Full CRUD on `id` (Computed, server-assigned), `title`, `description`, `callback_url` (wire field `callbackurl`), `version`, `developer_id`, `app_key`/`app_secret` (Computed; `app_secret` also `Sensitive`), and nested `attribute` blocks (`ToAttributes`). `app_key`/`app_secret` are generated by SAP and returned **only** in the Create response — `create-or-update-or-read-an-application-using-subscription-key-e2645b5.md` states outright that a `GET` "will not fetch app key and secret" — so both are `Computed` with `UseStateForUnknown` and are simply carried forward from state on every subsequent `Read`/`Update` rather than re-fetched. |
| `developerhub_developer` | `/api/1.0/registrations` (`DevPortal_RegisteringUsers_CF`) | Create registers the user as an administrator (`POST`); if Developer Hub already knows the user (`409`, e.g. access revoked earlier), access is granted again through the documented status update (`PUT` with `registered`). Read finds the user in the registered list; a `revoked` or `rejected` status counts as gone, so the next plan re-registers. Delete revokes access (`PUT` with `revoked` and the optional `revocation_reason`, which Developer Hub e-mails to the user) - it never deletes identity-provider users. The API cannot change name, e-mail or country, so those force replacement. |
| `developerhub_product_subscription` | `APIMgmt.Subscriptions` (OData, devportal, top-level entity set) | Full CRUD on `id` (Computed, server-assigned), `application_id` (`RequiresReplace` — no documented way to re-parent a subscription to a different application), `product_name` (in-place `Update`, confirmed by a worked `PUT` payload that changes `ToAPIProduct`/`product_id`), and `status` (Computed, read-only, opaque pass-through of SAP's own subscription lifecycle state — see §9). |

## 7. Data Source Mapping

| Terraform type | Backing API | Notes |
|---|---|---|
| `developerhub_current_user` | `GET /api/1.0/user` | Returns the identity of the credentials used to authenticate — useful to discover your own `developer_id` for `developerhub_application.developer_id`, exactly as the SAP doc describes obtaining it (§4/§11). |
| `developerhub_applications` | `GET /odata/1.0/data.svc/APIMgmt.Applications` (all pages) | Lists applications, optionally narrowed by `developer_id` (filtered client-side, so no OData `$filter` syntax is assumed). SAP's collection read does not return `app_key`/`app_secret`, so neither does this data source. |
| `developerhub_product_subscriptions` | `GET /odata/1.0/data.svc/APIMgmt.Subscriptions` (all pages) | Lists subscriptions with their status, optionally narrowed by `application_id` and/or `product_name` (client-side). Shows subscriptions developers created in the UI, which Terraform does not manage. |
| `developerhub_products` | `GET /odata/1.0/data.svc/APIMgmt.APIProducts` (all pages) | Published products with name (key), title, version, vendor, description, short text and publication data, optionally narrowed by `name`, so `product_name` values can be looked up. |
| `developerhub_registration_requests` | `GET /api/1.0/registrations?type=pending` | Users waiting for an administrator decision on their registration. |
| `developerhub_registered_users` | `GET /api/1.0/registrations?type=registered` | Lists registered Developer Hub developers and their `userId` (== `developer_id`), so a Product Subscription/Application configuration never needs an internal ID copied by hand from the UI. |

## 8. Terraform IDs

- `developerhub_application`: the Terraform ID is the SAP-assigned
  application id (the `id` field of `APIMgmt.Applications`, a GUID-shaped
  string). Import: `terraform import developerhub_application.example <id>`.
- `developerhub_product_subscription`: the Terraform ID is the SAP-assigned
  subscription id (the `id` field of the top-level `APIMgmt.Subscriptions`
  entity, server-assigned on create the same way an application's own `id`
  is). `application_id` and `product_name` are both plain fields on the
  entity itself (`app_id`, `product_id`) and so are populated by `Read`
  without needing to be encoded into the import identifier. Import:
  `terraform import developerhub_product_subscription.example <subscription_id>`.

Display names/titles are never used as Terraform IDs; SAP's own payloads use
stable technical identifiers (`id` for applications, the product's technical
`name` for the product reference), and this provider follows that.

## 9. Subscriptions: Design History and the One Remaining Inference

This section's design changed once during development, on real evidence,
and the history is kept here rather than erased because it is exactly the
kind of correction the project brief asks to be documented rather than
quietly fixed.

**First pass**: `custom-attributes-90a5a6d.md` shows a subscription nested
inside an `APIMgmt.Applications` create payload
(`"ToSubscriptions": [{"id": "...", "ToAPIProduct": [...]}]`), with no
worked example of reading, updating, or deleting one standalone. The first
implementation therefore *inferred* that a subscription must be addressed
as a keyed child of its application
(`APIMgmt.Applications('<app_id>')/ToSubscriptions('<subscription_id>')`),
by analogy with how `APIMgmt.Attributes` is addressed elsewhere in this API
family.

**Correction**: a later, more thorough pass through the same documentation
mirror turned up
`create-or-update-or-read-an-application-using-subscription-key-e2645b5.md`,
which turned out to hold the actual EDMX `<EntityType Name="SubscriptionsType">`
declaration and multiple full worked payloads. It shows that
`APIMgmt.Subscriptions` is in fact its **own top-level, independently
addressable entity set** (`GET`/`POST`/`PUT` directly against
`APIMgmt.Subscriptions('<id>')`, keyed by `id`), not a nested collection.
The entity carries plain `app_id`, `product_id`, `developer_id`,
`isSubscribed`, and `status` fields alongside its `ToApplication`/
`ToAPIProduct`/`ToRatePlan` navigation properties. A create sends the
relationship via `ToApplication`/`ToAPIProduct` navigation references
(`__metadata.uri`); a later `PUT` example that changes the subscribed
product sends the `product_id` field and the `ToAPIProduct` reference
together, which is exactly what `UpdateSubscriptionProduct`
(`internal/client/developerhub/subscriptions.go`) does. `RatePlans`
(`ToRatePlan`) is part of SAP's Monetization feature and is out of scope
here (§18/§19) — it is simply omitted from every request this provider
sends, which the same document confirms is valid for a non-monetized
subscription (the "older", non-Subscription-entity application flow never
sets a rate plan at all).

The **one inference that remains**: `DELETE` on `APIMgmt.Subscriptions`
itself is not shown verbatim anywhere found in this pass. It is inferred
from the sibling `APIMgmt.Applications` entity's documented
`DELETE .../APIMgmt.Applications('<app_id>')` on the exact same OData
service, using the exact same addressing convention now confirmed for
`GET`/`POST`/`PUT` on `APIMgmt.Subscriptions` itself — a much
higher-confidence inference than the design it replaced. If SAP's actual
behavior differs, `Delete` will surface that as a normal API error rather
than silently succeed, and the gap is tracked in `FEATURE_MATRIX.md`.

`setting-up-external-governance-992512e.md` additionally confirms that a
subscription rejected under External Governance is **deleted by SAP
automatically**, not left in a terminal "Rejected" state — this resource's
existing not-found handling on `Read` already covers that correctly with no
special-casing needed.

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
- **x509-based service keys** — the `devportal-apiaccess` plan supports
  certificate-based service keys in addition to `binding-secret`. Only
  `binding-secret` (client id/secret) is implemented in the first release.
- **Rate plans** (`APIMgmt.RatePlans`) — real, named in the `Subscriptions`
  EDMX (§9), but tied to SAP's Monetization feature, which the project
  brief does not ask for. Not implemented; a subscription is created
  without a `ToRatePlan` reference, which SAP's own documentation confirms
  is valid for a non-monetized subscription.

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

An unauthenticated request does not get a 401: the Developer Hub's
application router answers it with HTTP 200 and an HTML page that redirects
to the browser login (observed on a live tenant, 2026-09-26). The client
treats any `text/html` response as that case and reports that the request
was not authenticated and which provider settings to check, instead of
surfacing a JSON or XML parse error.

A token endpoint that rejects the client credentials (any `4xx` other than
`429` from `token_url`) is reported at once, naming `token_url`, `client_id`
and `client_secret`, instead of being retried like a transient failure:
retrying cannot fix credentials the identity service has refused.

## 15. Retry Strategy and Pagination

### Pagination

OData v2 services may page collections server-side, returning a `__next`
link with each page. `listAll` (`internal/client/developerhub/odata.go`)
follows it for every collection read (applications, subscriptions,
attributes), whether the link is absolute or relative. Because every
request carries the bearer token, a `__next` link to any host other than
the configured Developer Hub URL is refused, never followed, and the loop
stops with an error after 1000 pages rather than running forever.

### Retry

Copied near-verbatim from `terraform-provider-integration-suite`'s
`internal/client/http`: exponential backoff with full jitter on `429`,
`502`, `503`, `504`; `Retry-After` header honored when present; a single
retry after invalidating the cached token on `401`; CSRF token fetch-and
retry-once on a `403` carrying `X-CSRF-Token: Required`, which this API
family requires for write operations (seen in the `Attributes` sample
payloads). This logic is intentionally protocol-agnostic so it did not need
to change to fit Developer Hub's OData v1.0 service instead of Integration
Suite's OData v2 services.

Tokens are fetched lazily, on the first request that needs one, which is
after Terraform has already cancelled the context it passed to
`ConfigureProvider`. The token source therefore keeps that context's values
but not its cancellation (`context.WithoutCancel`); the per-request context
still bounds each API call.

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

## 18. Follow-Up Research Passes

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
  the UI - at the time, this was the only confirmation found that
  `developerhub_application` has generated credentials, and only as a UI
  screen, not a documented JSON API response.
  `configure-mcp-server-access-using-default-authentication-xsuaa-dc283ab.md`:
  MCP server "Agent subscriptions" also generate a client ID/secret through
  Developer Hub, but again only described as a UI flow ("Developer Hub
  automatically generates the client ID and client secret").

A **third** pass, prompted by continuing to grep the documentation mirror
more broadly (`grep -rn "api/1\.0\|/odata/1\.0" docs/`) rather than only
following the Developer Hub-labeled table of contents, found
`create-or-update-or-read-an-application-using-subscription-key-e2645b5.md`
filed under the classic API Management docs tree rather than the Developer
Hub one — which is exactly why the first two passes missed it. It resolved
what the second pass could not: it holds the actual EDMX and literal
request/response payloads confirming `app_key`/`app_secret`'s field names,
the `description`/`callbackurl` fields, and the entire corrected design of
`APIMgmt.Subscriptions` as a top-level entity documented in §9. Both are
now implemented; see §5's matrix and §6.

The lesson for anyone continuing this research: SAP's Help Portal mirror
groups Developer Hub content into more than one directory
(`ISuite_Integrations_APIs/` and `apim/API-Management/` both contain
Developer Hub-relevant pages, sometimes as literal duplicates and sometimes
not), so a keyword grep across the whole mirror finds pages a
table-of-contents walk of one directory alone will miss.

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

## 20. API Verification Policy

Every feature must be built against, and re-validated against, the **newest**
version of each of these sources, in this order of authority:

1. **The live Developer Hub tenant.** For the OData service
   (`/odata/1.0/data.svc/`), the OData protocol requires every service to
   publish its current contract at `$metadata`. `developerhub.Contract`
   (`internal/client/developerhub/contract.go`) lists every entity set,
   property and navigation property the provider reads or writes, and
   `TestAccServiceContract` checks it against the tenant's live `$metadata`
   on every acceptance run, failing with the exact entity set and field if
   SAP renames or removes one. `TestContract_CoversEveryWireField` (a plain
   unit test) derives field names from the client's wire structs by
   reflection, so a field cannot be added to the client without also being
   added to the contract. The `/api/1.0/` endpoints are plain JSON rather
   than OData, so they are covered by the data sources' own acceptance tests
   instead. Run all of this with `make verify-api`.
2. **SAP Business Accelerator Hub (`api.sap.com`) OpenAPI specs** for the
   `APIMgmt` package's Developer Hub artifacts.
3. **SAP Help Portal (`help.sap.com`).**

### Current verification status

| Source | Version checked | Status |
|---|---|---|
| `help.sap.com` | mirror commit `33f3395`, 2026-09-18 — re-checked 2026-09-25 with direct network access | **Verified.** The live site is a single-page app whose page code loads each topic's history from SAP's GitHub documentation repositories, so the `SAP-docs/btp-integration-suite` mirror at its newest commit *is* the current `help.sap.com` content. Every endpoint and field in `Contract` traces to a quoted payload or EDMX in this revision (§5, §9). The API Management "What's New" list (`what-s-new-for-sap-api-management-cloud-foundry-d9d60be.md`, 4 329 lines) was reviewed in full for Developer Hub entries: it announces UI features only (MCP server subscriptions, product permissions, AsyncAPI re-upload, external OAuth products, centralized Developer Hub certificate monitoring) and no new public Developer Hub REST API, which matches §5 and §12. |
| `api.sap.com` specs | `DevPortal_Application_CF` 1.0, `DevPortal_RegisteringUsers_CF` 1.0, `DevPortal_ExternalGovernance_CF` 1.0.0, `DevPortal_ExternalGovernanceSPI_CF` 1.0.0 — downloaded 2026-09-26, kept in `api-specs/` | **Verified.** Downloads require an interactive SAP Universal ID sign-in, so the files were downloaded in a browser and committed. Findings: the OData service wraps responses in `d` / `d.results` (as `decodeEnvelope` expects); every Application, Subscription and product field in `Contract` exists; three attribute details differ from the older Help Portal sample and now follow the spec (§5); the registration API is fully specified and is now implemented (`developerhub_developer`, `developerhub_registration_requests`); the product type is fully specified and is now implemented (`developerhub_products`). The application spec covers only get-one, create and the attribute operations: application update, delete and collection read, and all `APIMgmt.Subscriptions` operations, remain sourced from the Help Portal (§5, §9). The external governance specs contain only the decision and deletion-confirmation callbacks (§5). |
| community.sap.com (corroborating, lower authority) | SAP-authored posts: "Exploring API Portal and API Business Hub Enterprise APIs" (2022), "Governance in Developer Hub" parts 1-3 (2025-12-17) | **Consistent with §5.** Confirms `/odata/1.0/data.svc/$metadata` as the way to discover the service contract, the placeholder `id` on create, `developer_id` being optional, and that the service exposes the entity sets `APIMgmt.Applications`, `APIMgmt.APIProducts`, `APIMgmt.APIResources`, `APIMgmt.ProxyEndPoints`, `APIMgmt.APIProxies`, `APIMgmt.APIResourceDocumentations`, `APIMgmt.RatePlans`, `APIMgmt.Subscriptions`, `APIMgmt.Attributes`, `CatalogResources`, `Comments`, `Ratings`. Governance settings are confirmed UI-only; the External Governance API's two resources are named (see §5). |
| Live Developer Hub `$metadata` and `/api/1.0/` | tenant on `cfapps.eu10-003`, first `make verify-api` run 2026-09-26 | **Blocked on credentials.** The host answers; without a token every call returns the login redirect described in §14. The first run with `SAP_DEVELOPER_HUB_*` set found a provider bug (token fetches used the already-cancelled configure context, fixed, §15) and then stopped at the token endpoint, which rejected the configured client credentials with HTTP 401 (`unauthorized`). The API itself was therefore not reached yet; the next run needs the `clientid`/`clientsecret` of a `devportal-apiaccess` service key (§4). |

Update this table whenever any of the three is checked again.
