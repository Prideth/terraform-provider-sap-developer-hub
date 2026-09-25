# Users waiting for an administrator to approve their Developer Hub
# registration.
data "developerhub_registration_requests" "pending" {}

output "pending_users" {
  value = [for r in data.developerhub_registration_requests.pending.request : r.email]
}
