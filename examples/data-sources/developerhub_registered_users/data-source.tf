data "developerhub_registered_users" "all" {}

output "developer_ids" {
  value = [for u in data.developerhub_registered_users.all.user : u.user_id]
}
