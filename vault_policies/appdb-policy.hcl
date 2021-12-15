path "secret/data/mongodbenterprise/appdb/*" {
  capabilities = ["read", "list"]
}
path "secret/metadata/mongodbenterprise/appdb/*" {
  capabilities = ["list"]
}
