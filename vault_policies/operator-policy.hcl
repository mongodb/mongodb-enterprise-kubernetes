path "secret/data/mongodbenterprise/*" {
  capabilities = ["create", "read", "update", "delete", "list"]
}
path "secret/metadata/mongodbenterprise/*" {
  capabilities = ["list", "read"]
}
