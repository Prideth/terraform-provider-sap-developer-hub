# Feature Matrix

See `DESIGN.md` §5 for the full API research behind this table, including
citations for every "verified" claim.

| Feature | Resource | Data Source | Import | Status |
|---|---|---|---|---|
| Applications | `developerhub_application` | `developerhub_applications` | yes | supported |
| Application custom attributes | nested in `developerhub_application` | – | via parent | supported |
| Product subscriptions | `developerhub_product_subscription` | `developerhub_product_subscriptions` | yes | supported (verified create/read/update; delete inferred by analogy, see `DESIGN.md` §9) |
| Current user / developer identity | – | `developerhub_current_user` | n/a | supported |
| Registered developers | – | `developerhub_registered_users` | n/a | supported |
| Bulk user registration (admin) | – | – | – | SAP API unavailable to verify (`DevPortal_RegisteringUsers_CF` exists but schema unconfirmed) |
| Domain categories | – | – | – | SAP API unavailable (no documented public REST API found) |
| Access governance (site visibility) | – | – | – | SAP API unavailable (no documented public REST API found) |
| Subscription governance settings | – | – | – | SAP API unavailable (no documented public REST API found) |
| External governance (SPI destination) | – | – | – | out of scope (BTP Cockpit destination configuration) |
| Products / API bundles | – | – | – | out of scope — owned by `terraform-provider-integration-suite` |
| Rate plans (Monetization) | – | – | – | out of scope — entity confirmed to exist (`APIMgmt.RatePlans`), Monetization not requested by the project brief |
| MCP Server products | – | – | – | out of scope — product authoring belongs to `terraform-provider-integration-suite`; no separate Developer Hub API found |
| Business systems / content discovery | – | – | – | SAP API unavailable / likely belongs to BTP system landscape tooling |
| Site configuration / branding | – | – | – | SAP API unavailable (Site Editor is UI-only) |
| Notifications | – | – | – | SAP API unavailable (UI-only) |
| Centralized Developer Hub connections | – | – | – | SAP API unavailable on the Developer Hub side; action is explicitly irreversible |
| Product discovery/subscription permissions (custom role collections) | – | – | – | out of scope — BTP custom role collections |
