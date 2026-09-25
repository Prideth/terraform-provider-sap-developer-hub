data "developerhub_current_user" "me" {}

# All applications of the developer the provider authenticates as.
data "developerhub_applications" "mine" {
  developer_id = data.developerhub_current_user.me.name
}

output "application_titles" {
  value = [for a in data.developerhub_applications.mine.application : a.title]
}
