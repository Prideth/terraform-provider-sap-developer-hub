data "developerhub_products" "catalog" {}

# Subscribe to a product by looking it up rather than hard-coding its name.
data "developerhub_products" "sales" {
  name = "Sales_API"
}

output "product_titles" {
  value = { for p in data.developerhub_products.catalog.product : p.name => p.title }
}
