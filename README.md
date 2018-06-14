# MongoDB Enterprise Kubernetes Operator #

Welcome to the MongoDB Enterprise Kubernetes Operator. The Operator enables easy deploys of MongoDB into Kubernetes clusters, using our management, monitoring and backup platforms, Ops Manager and Cloud Manager. By installing this integration, you will be able to deploy MongoDB instances with a single simple command.

Please note that this project is currently in beta, and is not yet recommended for production use.

You can discuss this integration in our [Slack](https://community-slack.mongodb.com) - join the [#enterprise-kubernetes](https://mongo-db.slack.com/messages/CB323LCG5/) channel.

## Requirements ##

The MongoDB Enterprise Operator is compatible with Kubernetes v1.9 and above. It has been tested against Openshift 3.9.

This Operator requires Ops Manager or Cloud Manager. In this document, when we refer to "Ops Manager", Cloud Manager may also be used.


## Helm Installation ##

If you have an Helm installation in your Kubernetes cluster, you can run:

    helm install helm_chart/ --name mongodb-enterprise


## Non-Helm install ##

This operator can also be installed using yaml files, in case you are not using Helm.

    kubectl apply -f mongodb-enterprise.yaml


## Adding Ops Manager Credentials ##

For the Operator to work, you will need the following information:

* Base Url - the url of an Ops Manager instance
* Project Id - the id of a Project which MongoDBs will be deployed into.
* User - an Ops Manager username
* Public API Key - an Ops Manager Public API Key. Note that you must whitelist the IP range of your Kubernetes cluster so that the Operator may make requests to Ops Manager using this API Key.

### Projects ###

A `Project` object is a Kubernetes `ConfigMap` that points to an Ops Manager installation and a `Project`. This `ConfigMap` has the following structure:


```

$ cat my-project.yaml
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-project
  namespace: mongodb
data:
  projectId: my-project-id
  baseUrl: https://my-ops-cloud-manager-url
```

Apply this file to create the new `Project`:

    kubectl apply -f my-project.yaml

### Credentials ###

For a user to be able to create or update objects in this Ops Manager Project they need a Public API Key. These will be held by Kubernetes as a `Secret` object. You can create this Secret with the following command:

``` bash
$ kubectl -n mongodb create secret generic my-credentials --from-literal="user=some@example.com" --from-literal="publicApiKey=my-public-api-key"
```

In this example, a `Secret` object with the name `my-credentials` was created. The contents of this `Secret` object is the `user` and `publicApiKey` attribute. You can see this secret with a command like:

``` bash
$ kubectl describe secrets/my-credentials -n mongodb

Name:         my-credentials
Namespace:    mongodb
Labels:       <none>
Annotations:  <none>

Type:  Opaque

Data
====
publicApiKey:  41 bytes
user:          14 bytes
```

We can't see the contents of the `Secret`, because it is a secret!
This is good, it will allow us to maintain a separation between our
users.

### Creating MongoDB Object ###

A MongoDB object in Kubernetes can be a Standalone, a Replica Set or a Sharded Cluster. We are going to create a Replica Set to test that everything is working as expected. There is a ReplicaSet creation yaml file in the `samples/` directory. The contents of this file are as follows:

``` yaml
---
apiVersion: mongodb.com/v1
kind: MongoDbReplicaSet
metadata:
  name: my-replica-set
  namespace: mongodb
spec:
  members: 3
  version: 3.6.5

  persistent: false  # For testing, create Pods with NO persistent volumes.

  project: my-project
  credentials: my-credentials

```

If you have a correctly created Project with the name `my-project` and Credentials stored in a secret called `my-credentials` then, after applying this file then everything should be running now and a new Replica Set with 3 members should soon appear in Ops Manager UI.


    kubectl apply -f samples/replicaset.yaml
