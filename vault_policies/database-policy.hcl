path "secret/data/mongodbenterprise/database/*" {
  capabilities = ["read", "list"]
}
path "secret/metadata/mongodbenterprise/database/*" {
  capabilities = ["list"]
}
