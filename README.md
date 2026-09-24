# Terraform Provider for SAP Integration Suite Developer Hub

A Terraform provider for the **SAP Integration Suite Developer Hub** - the
catalog application developers use to discover and subscribe to the APIs,
Events and MCP servers an organization publishes.

## Status

Early, actively developed. Two resources and two data sources are
implemented, each against a Developer Hub API endpoint independently
verified from SAP's own documentation (see [`DESIGN.md`](DESIGN.md) §5).
Nothing in this provider is implemented against a guessed or reverse
engineered endpoint - see [Known limitations](#known-limitations) for what
is deliberately not yet supported, and why.

## Scope

This provider manages **Developer Hub catalog and consumer-lifecycle
configuration** for a Developer Hub that is already activated and
reachable: applications, their custom attributes, and their product
subscriptions.

It does **not** provision SAP BTP itself (global accounts, subaccounts,
Cloud Foundry spaces, entitlements, role collections) - use the official
[`SAP/btp`](https://registry.terraform.io/providers/SAP/btp/latest)
provider for that.

It also does **not** activate the Developer Hub capability, and does not
author or publish Products/APIs - those are the responsibility of
[`terraform-provider-integration-suite`](https://github.com/Prideth/terraform-provider-integration-suite),
which this provider is designed to be used alongside. If Developer Hub is
not reachable for the configured tenant, every resource and data source in
this provider fails with a clear message telling you to activate it through
the Integration Suite provider first, rather than trying to activate it
itself.

See [`DESIGN.md`](DESIGN.md) for the full scope boundary, the SAP API
research behind every resource, and every feature that was investigated but
is **not** implemented because SAP does not currently expose a documented
public API for it.

## Feature Support

See [`FEATURE_MATRIX.md`](FEATURE_MATRIX.md) for the complete list.

| Feature | Resource | Data source | Import |
|---|---|---|---|
| Applications | `developerhub_application` | – | yes |
| Application custom attributes | nested in `developerhub_application` | – | via parent |
| Product subscriptions | `developerhub_product_subscription` | – | yes |
| Current user / developer identity | – | `developerhub_current_user` | n/a |
| Registered developers | – | `developerhub_registered_users` | n/a |

Domain categories, access/subscription governance settings, site
configuration, notifications and centralized Developer Hub connections were
all researched (`DESIGN.md` §5/§12) and have no currently documented public
REST API, so they are not implemented rather than built against a guess.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.5
- Go >= 1.24 (for building the provider from source)
- A reachable Developer Hub with a `devportal-apiaccess` service instance
  and service key (see [Authentication](#authentication))

## Installation

```hcl
terraform {
  required_providers {
    developerhub = {
      source  = "Prideth/sap-developer-hub"
      version = "~> 0.1"
    }
  }
}
```

## Authentication

The provider authenticates with an OAuth2 client-credentials grant, using
the credentials from a Developer Hub `devportal-apiaccess` service key (see
`DESIGN.md` §4 for how to create one). Configure it either via provider
attributes:

```hcl
provider "developerhub" {
  url           = "https://<devportal-application-url>"
  token_url     = "https://<token-endpoint>/oauth/token"
  client_id     = var.client_id
  client_secret = var.client_secret
}
```

or via environment variables, which is the recommended way to avoid
committing credentials to a `.tf` file:

| Environment variable | Corresponds to |
|---|---|
| `SAP_DEVELOPER_HUB_URL` | `url` |
| `SAP_DEVELOPER_HUB_TOKEN_URL` | `token_url` |
| `SAP_DEVELOPER_HUB_CLIENT_ID` | `client_id` |
| `SAP_DEVELOPER_HUB_CLIENT_SECRET` | `client_secret` |

`client_secret` is always treated as sensitive: it is never logged, never
included in a diagnostic, and never printed in debug output.

## Example

```hcl
data "developerhub_current_user" "me" {}

resource "developerhub_application" "sales_app" {
  title        = "Sales Application"
  developer_id = data.developerhub_current_user.me.name

  attribute {
    name  = "environment"
    value = "production"
  }
}

# "Sales_API" is a product already published from SAP Integration Suite /
# API Portal - authoring it is out of scope for this provider (see Scope).
resource "developerhub_product_subscription" "sales_app_to_sales_api" {
  application_id = developerhub_application.sales_app.id
  product_name   = "Sales_API"
}
```

More examples, including a realistic multi-application setup, are under
[`examples/`](examples/).

## Import

Both resources support `terraform import`:

```shell
terraform import developerhub_application.sales_app <application_id>
terraform import developerhub_product_subscription.sales_app_to_sales_api <application_id>/<subscription_id>
```

See each resource's own documentation under [`docs/resources/`](docs/resources/)
for the exact ID format.

## Development

```shell
make build       # go build
make fmt          # gofmt -w .
make vet          # go vet ./...
make lint         # golangci-lint run ./... (requires golangci-lint)
make docs         # regenerate docs/ with tfplugindocs
```

## Testing

```shell
make unit-test          # go test -race -cover ./... - always runs, no credentials needed
make acceptance-test     # TF_ACC=1 go test -v -timeout 60m ./... - requires real credentials
```

Acceptance tests exercise the full create/plan/update/import cycle of each
resource against a real Developer Hub tenant. They require
`TF_ACC=1` plus the same `SAP_DEVELOPER_HUB_*` environment variables the
provider itself reads, and the product subscription test additionally
requires `SAP_DEVELOPER_HUB_TEST_PRODUCT_NAME` (the technical name of an
already-published product in the test tenant, since this provider does not
author products - see Scope). Every acceptance test skips itself
gracefully, rather than failing, when these are not set. No credentials are
ever committed to this repository.

## Releases

Tagged releases (`vX.Y.Z`, following [SemVer](https://semver.org/)) are
built and published by [GoReleaser](https://goreleaser.com/) via
`.github/workflows/release.yml`, producing signed, checksummed binaries for
linux/windows/darwin on amd64/arm64, plus the
`terraform-registry-manifest.json` the Terraform Registry requires.

## Known limitations

See [`DESIGN.md`](DESIGN.md) §12 for the full, cited list. In short: this
provider does not manage domain categories, access governance, subscription
governance settings, site configuration/branding, notifications, or
centralized Developer Hub connections, because no currently documented
public REST API for them was found. Application `app_key`/`app_secret` are
confirmed to exist by SAP's UI documentation but are not exposed here
because no JSON sample shows their actual field names.

## Disclaimer

This is an independent, community-maintained project. It is not developed,
endorsed, or supported by SAP SE.

## License

[Apache License 2.0](LICENSE)
