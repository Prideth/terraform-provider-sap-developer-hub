data "developerhub_current_user" "me" {}

output "developer_id" {
  value = data.developerhub_current_user.me.name
}
