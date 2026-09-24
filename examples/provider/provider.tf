terraform {
  required_providers {
    developerhub = {
      source  = "Prideth/sap-developer-hub"
      version = "~> 0.1"
    }
  }
}

# Credentials can also be supplied entirely through environment variables
# (SAP_DEVELOPER_HUB_URL, SAP_DEVELOPER_HUB_TOKEN_URL,
# SAP_DEVELOPER_HUB_CLIENT_ID, SAP_DEVELOPER_HUB_CLIENT_SECRET), which is
# the recommended way to avoid committing credentials to a .tf file.
provider "developerhub" {
  url           = var.developer_hub_url
  token_url     = var.developer_hub_token_url
  client_id     = var.developer_hub_client_id
  client_secret = var.developer_hub_client_secret
}
