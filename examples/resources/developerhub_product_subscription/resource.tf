# "Sales_API" is the technical name of a product already published from
# SAP Integration Suite / API Portal - authoring the product itself is out
# of scope for this provider (see the provider's DESIGN.md §2); it is
# managed with terraform-provider-integration-suite instead.
resource "developerhub_product_subscription" "sales_app_to_sales_api" {
  application_id = developerhub_application.sales_app.id
  product_name   = "Sales_API"
}
