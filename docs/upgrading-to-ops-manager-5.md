# Upgrading to Ops Manager 5.0.0

In Ops Manager 5.0.0 the old-style _Personal API Keys_ have been deprecated. In
order to upgrade to Ops Manager 5.0.0 a new set of _Programmatic API Keys_ will
have to be generated, before the upgrade process.

In this document we explain how to generate a _Programmatic API Key_ using
[mongocli](https://docs.mongodb.com/mongocli/stable/).

## Obtain your current credentials

The credentials used by the Operator are stored in a `Secret`. This secret will
be in the same `Namespace` as the Operator and the name will have the following
structure:

```
<namespace>-<ops-manager-resource-name>-admin-key
```

We can fetch the current credentials in order to use them later with the
following code snippet, make sure you set `om_resource_name` and `namespace` to
the correct values:

```sh
om_resource_name="ops-manager-resource-name"
namespace="ops-manager-resource-namespace"
public_key=$(kubectl get "secret/${namespace}-${resource_name}-admin-key" -o jsonpath='{.data.user}' | base64 -d)
private_key=$(kubectl get "secret/${namespace}-${resource_name}-admin-key" -o jsonpath='{.data.publicApiKey}' | base64 -d)
```

## Using `mongocli`

We will use [mongocli](https://docs.mongodb.com/mongocli/stable/) to create the
different resources we need: a _whitelist_ and a _programmatic API key_.

Make sure you visit `mongocli` and install it before proceeding.

### Configuring `mongocli`

To configure `mongocli` to be able to talk to Ops Manager:

```
mongocli config --service ops-manager
```

`mongocli` will ask for your _Ops Manager URL_, _Public API Key_ and _Private
API Key_. Use the ones you fetched in the previous command, they have been
stored in the `public_key` and `private_key` variables for you.

### Create a Global Access List

We first need to start creating a [Global Access
List](https://docs.mongodb.com/mongocli/stable/command/mongocli-iam-globalAccessLists-create/#std-label-mongocli-iam-globalAccessLists-create)

```
mongocli iam globalAccessLists create --cidr "<your-ip-in-cidr-notation>" --desc "Our first range of allowed IPs"
```

You can add as many as you need for your organization. For instance, to allow
access from all the Kubernetes private network:

```
mongocli iam globalAccessLists create --cidr "10.0.0.0/8" --desc "Allow access from internal network."
```

- Please note: some clusters might use a different network configuration.
  Consult you Kubernetes provider or administrator to find out the correct
  configuration for your Kubernetes cluster.

### Create a Programmatic API Key

A new [Programmatic API
Key](https://docs.mongodb.com/mongocli/stable/command/mongocli-iam-globalApiKeys-create/#std-label-mongocli-iam-globalApiKeys-create)
needs to be created, we will also use `mongocli` for this:

```
mongocli iam globalApiKeys create --role GLOBAL_OWNER --desc "New API Key for the Kubernetes Operator"
```

The output of this command will be similar to:

```json
{
  "id": "60ed976ec409d34da670bffe",
  "desc": "New programmatic API key for the operator",
  "roles": [
    {
      "roleName": "GLOBAL_OWNER"
    }
  ],
  "privateKey": "1980bd92-f81a-41e1-b302-a2308fcc450a",
  "publicKey": "dhjjpfgf"
}
```

Make sure you write down the `privateKey` and `publicKey`. It is not possible to
recover the `privateKey` part at a later stage.

## Configure the Operator to use the new credentials

Edit the _API Key_ `Secret` in place with `kubectl edit` or apply the following
yaml segment:

```sh
cat <<EOF | kubectl apply -f -
---
kind: Secret
apiVersion: v1
type: Opaque
metadata:
  name: <om-namespace>-<om-name>-admin-key
  namespace: <om-namespace>
stringData:
  publicApiKey: "<your-newly-created-private-api-key>"
  user: "<your-newly-created-public-api-key>"
EOF
```

- Please note that the returned `publicKey` corresponds to the `user` entry and
  the `privateKey` corresponds to the `publicApiKey`.

# Proceed with the Upgrade

After the API Key has been changed to a new _Programmatic API Key_ it will be
possible to upgrade the `MongoDBOpsManager` resource to latest version: 5.0.0.
