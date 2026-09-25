# Everyone subscribed to a product, including subscriptions developers
# created themselves in the Developer Hub UI, with their approval status.
data "developerhub_product_subscriptions" "sales_api" {
  product_name = "Sales_API"
}

output "pending_subscriptions" {
  value = [
    for s in data.developerhub_product_subscriptions.sales_api.subscription : s.application_id
    if !s.is_subscribed
  ]
}
