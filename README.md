# MongoDB Enterprise Kubernetes Operator #

Welcome to the MongoDB Enterprise Kubernetes Operator. The Operator enables easy deploys of MongoDB into Kubernetes clusters, using our management, monitoring and backup platforms, Ops Manager and Cloud Manager. By installing this integration, you will be able to deploy MongoDB instances with a single simple command.

Please note that this project is currently in beta, and is not yet recommended for production use.

You can discuss this integration in our [Slack](https://community-slack.mongodb.com) - join the [#enterprise-kubernetes](https://mongo-db.slack.com/messages/CB323LCG5/) channel.

## Documentation ##

[Install Kubernetes Operator](https://docs.opsmanager.mongodb.com/current/tutorial/install-k8s-operator)

[Deploy Standalone](https://docs.opsmanager.mongodb.com/current/tutorial/deploy-standalone)

[Deploy Replica Set](https://docs.opsmanager.mongodb.com/current/tutorial/deploy-replica-set)

[Deploy Sharded Cluster](https://docs.opsmanager.mongodb.com/current/tutorial/deploy-sharded-cluster)

[Edit Deployment](https://docs.opsmanager.mongodb.com/current/tutorial/edit-deployment)

[Kubernetes Resource Specification](https://docs.opsmanager.mongodb.com/current/reference/k8s-operator-specification)

[Known Issues for Kubernetes Operator](https://docs.opsmanager.mongodb.com/current/reference/known-issues-k8s-beta)

## Requirements ##

The MongoDB Enterprise Operator is compatible with Kubernetes v1.9 and above. It has been tested against Openshift 3.9.

This Operator requires Ops Manager or Cloud Manager. In this document, when we refer to "Ops Manager", you may substitute "Cloud Manager". The functionality is the same.



## Installation

### Create Kubernetes Namespace

The Mongodb Enterprise Operator is installed, by default, into the `mongodb` Namespace, but this Namespace is not created automatically. To create this Namespace you should execute:

    kubectl create namespace mongodb
    
If you plan on using any other Namespace, please make sure you update the yaml files' `metadata.namespace` attribute to 
point to your preferred Namespace. If using `helm` you need to override the `namespace` attribute with `--set namespace=<..>` 
during helm installation

### Installation using yaml files

#### Create CustomResourceDefinitions

The `CustomResourceDefinition` (or `crds`) should be installed before installing the operator into your Kubernetes cluster. To do this, make sure you have logged into your Kubernetes cluster and that you can perform Cluster level operations:

    kubectl apply -f https://raw.githubusercontent.com/mongodb/mongodb-enterprise-kubernetes/master/crds.yaml

This will create three new `crds` in your cluster, `MongoDbStandalone`, `MongoDbReplicaSet` and `MongoDbShardedCluster`. These new objects will be the ones used by the operator to perform the MongoDb operations needed to prepare each one of the different MongoDb types of deployments.

#### Operator Installation

This operator can also be installed using yaml files, in case you are not using Helm. You may apply the config directly from github clone this repo, and apply the file

    kubectl apply -f https://raw.githubusercontent.com/mongodb/mongodb-enterprise-kubernetes/master/mongodb-enterprise.yaml

or clone this repo, make any edits you need, and apply it from your machine.

    kubectl apply -f mongodb-enterprise.yaml


### Helm Installation

If you have an Helm installation in your Kubernetes cluster, you can run:

    helm install helm_chart/ --name mongodb-enterprise



## Adding Ops Manager Credentials ##

For the Operator to work, you will need the following information:

* Base Url - the url of an Ops Manager instance
* Project Name - the name of an Ops Manager Project where MongoDBs will be deployed into. It will be created by Operator
 if it doesn't exist (and this is the recommended way instead of reusing the project created in OpsManager directly)
* (optionally) Organization Id - the id of organization to which Project belongs
* User - an Ops Manager username
* Public API Key - an Ops Manager Public API Key. Note that you must whitelist the IP range of your Kubernetes cluster so that the Operator may make requests to Ops Manager using this API Key.

This is documented in greater detail in our [installation guide](https://docs.opsmanager.mongodb.com/current/tutorial/install-k8s-operator)


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
  projectName: myProjectName
  orgId: 5b890e0feacf0b76ff3e7183 # this is an optional parameter
  baseUrl: https://my-ops-manager-or-cloud-manager-url
```
> Note, that if `orgId` is skipped then the new organization named `projectName` will be automatically created and new 
project will be added there.
 
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

### Creating a MongoDB Object ###

A MongoDB object in Kubernetes can be a MongoDBStandalone, a MongoDBReplicaSet or a MongoDBShardedCluster. We are going to create a replica set to test that everything is working as expected. There is a MongoDBReplicaSet yaml file in `samples/minimal/replicaset.yaml`.

If you have a correctly created Project with the name `my-project` and Credentials stored in a secret called `my-credentials` then, after applying this file then everything should be running and a new Replica Set with 3 members should soon appear in Ops Manager UI.

    kubectl apply -f samples/minimal/replicaset.yaml
