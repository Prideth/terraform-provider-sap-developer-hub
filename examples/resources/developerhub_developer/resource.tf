# Onboards a user of the subaccount's identity provider as an application
# developer, the same as "Add User" in the Developer Hub admin center.
resource "developerhub_developer" "jane" {
  user_id    = "jane.doe"
  email      = "jane.doe@example.com"
  first_name = "Jane"
  last_name  = "Doe"
  country    = "DE"

  # Sent to Jane by e-mail when this resource is destroyed and her access is
  # revoked.
  revocation_reason = "Left the partner integration team"
}

# The registered user can own applications right away.
resource "developerhub_application" "jane_app" {
  title        = "Jane's integration"
  developer_id = developerhub_developer.jane.user_id
}
