data "developerhub_current_user" "me" {}

resource "developerhub_application" "sales_app" {
  title        = "Sales Application"
  description  = "Used by the sales integration team"
  callback_url = "https://sales.example.com/oauth/callback"
  developer_id = data.developerhub_current_user.me.name

  attribute {
    name  = "environment"
    value = "production"
  }

  attribute {
    name  = "cost_center"
    value = "CC-4711"
  }
}
