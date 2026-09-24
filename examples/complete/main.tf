# A realistic example: SAP Integration Suite / API Portal (managed with
# terraform-provider-integration-suite, not shown here) has already
# published a handful of products. This configuration onboards an
# application for a partner integration team and subscribes it to the
# products it needs, using this provider only.

terraform {
  required_providers {
    developerhub = {
      source  = "Prideth/sap-developer-hub"
      version = "~> 0.1"
    }
  }
}

provider "developerhub" {
  # url, token_url, client_id and client_secret are supplied via
  # SAP_DEVELOPER_HUB_* environment variables - see the provider docs.
}

data "developerhub_registered_users" "all" {}

locals {
  # Look up the partner integration team's already-registered developer_id
  # instead of hardcoding an internal id copied from the Developer Hub UI.
  partner_developer_id = one([
    for u in data.developerhub_registered_users.all.user : u.user_id
    if u.email == "partner-integration-team@example.com"
  ])
}

resource "developerhub_application" "partner_integration" {
  title        = "Partner Integration"
  developer_id = local.partner_developer_id

  attribute {
    name  = "environment"
    value = "production"
  }

  attribute {
    name  = "partner"
    value = "contoso"
  }
}

resource "developerhub_product_subscription" "partner_sales_api" {
  application_id = developerhub_application.partner_integration.id
  product_name   = "Sales_API"
}

resource "developerhub_product_subscription" "partner_order_events" {
  application_id = developerhub_application.partner_integration.id
  product_name   = "Order_Events"
}

output "partner_application_id" {
  value = developerhub_application.partner_integration.id
}
