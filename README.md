# MongoDB Enterprise Kubernetes Operator #

Welcome to the MongoDB Enterprise Kubernetes Operator. The Operator enables easy deploys of MongoDB into Kubernetes clusters, using our management, monitoring and backup platforms, Ops Manager and Cloud Manager. By installing this integration, you will be able to deploy MongoDB instances with a single simple command.

Also the Operator allows to deploy Ops Manager into Kubernetes. Note, that currently this feature is **alpha**. See more information below.

You can discuss this integration in our [Slack](https://community-slack.mongodb.com) - join the [#enterprise-kubernetes](https://mongo-db.slack.com/messages/CB323LCG5/) channel.

## Documentation ##

[Install Kubernetes Operator](https://docs.opsmanager.mongodb.com/current/tutorial/install-k8s-operator)

[Deploy Standalone](https://docs.opsmanager.mongodb.com/current/tutorial/deploy-standalone)

[Deploy Replica Set](https://docs.opsmanager.mongodb.com/current/tutorial/deploy-replica-set)

[Deploy Sharded Cluster](https://docs.opsmanager.mongodb.com/current/tutorial/deploy-sharded-cluster)

[Edit Deployment](https://docs.opsmanager.mongodb.com/current/tutorial/edit-deployment)

[Kubernetes Resource Specification](https://docs.opsmanager.mongodb.com/current/reference/k8s-operator-specification)

[Troubleshooting Kubernetes Operator](https://docs.opsmanager.mongodb.com/current/reference/troubleshooting/k8s/)

[Known Issues for Kubernetes Operator](https://docs.opsmanager.mongodb.com/current/reference/known-issues-k8s-beta)

## Requirements ##

The MongoDB Enterprise Operator is compatible with Kubernetes v1.13 and above. It has been tested against Openshift 3.11.

This Operator requires [Ops Manager](https://docs.opsmanager.mongodb.com/current/) or [Cloud Manager](https://cloud.mongodb.com/user#/cloud/login). In this document, when we refer to "Ops Manager", you may substitute "Cloud Manager". The functionality is the same.
> If this is your first time trying the Operator, Cloud Manager is easier to get started


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

This will create a new `crd` in your cluster, `MongoDB`. This new object will be the one used by the operator to perform the MongoDb operations needed to prepare each one of the different MongoDb types of deployments.

#### Operator Installation

This operator can also be installed using yaml files, in case you are not using Helm. You may apply the config directly from github clone this repo, and apply the file

    kubectl apply -f https://raw.githubusercontent.com/mongodb/mongodb-enterprise-kubernetes/master/mongodb-enterprise.yaml

or clone this repo, make any edits you need, and apply it from your machine.

    kubectl apply -f mongodb-enterprise.yaml

Check the end of the page for instructions on how to remove the Operator.

### Installation using Helm Chart

If you have installed the Helm client locally then you can run (note that `helm install` is a less preferred way as makes upgrades more complicated.
`kubectl apply` is a much clearer way of installing/upgrading):

    helm template public/helm_chart > operator.yaml
    kubectl apply -f operator.yaml

You can customize installation by simple overriding of helm variables, for example use `--set operator.env="dev"` to run the Operator in development mode
(this will turn logging level to `Debug` and will make logging output as non-json)

Check the end of the page for instructions on how to remove the Operator.

## MongoDB object ##

*This section describes how to create the MongoDB resource. Follow the next section on how to work with Ops Manager resource.* 

### Adding Ops Manager Credentials ###

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

### Creating a MongoDB Object ###

A MongoDB object in Kubernetes is a MongoDB (short name `mdb`). We are going to create a replica set to test that everything is working as expected. There is a MongoDB replica set yaml file in `samples/minimal/replicaset.yaml`.

If you have a correctly created Project with the name `my-project` and Credentials stored in a secret called `my-credentials` then, after applying this file then everything should be running and a new Replica Set with 3 members should soon appear in Ops Manager UI.

    kubectl apply -f samples/minimal/replicaset.yaml

## Ops Manager object (alpha) ##

This section describes how to create the Ops Manager object in Kubernetes. Note, that this requires all 
the CRDs and the Operator application to be installed as described above.

*Disclaimer: this is an early release of Ops Manager - so it's not recommended to use it in production*

### Create Admin Credentials Secret ###

Before creating the Ops Manager object you need to prepare the information about the admin user which will be 
created automatically in Ops Manager. You can use the following command to do it:

```bash
$ kubectl create secret generic ops-manager-admin-secret  --from-literal=Username="jane.doe@example.com" --from-literal=Password="Passw0rd."  --from-literal=FirstName="Jane" --from-literal=LastName="Doe" -n <namespace>
```

Note, that the secret is needed only during the initialization of the Ops Manager object - you can remove it or 
clean the password field after the Ops Manager object was created

### Create Ops Manager object ###

Use the file `samples/ops-manager/ops-manager.yaml`. Edit the fields and create the object in Kubernetes:

```bash
$ kubectl apply -f samples/ops-manager/ops-manager.yaml
```

Note, that it takes up to 8 minutes to initialize the Application Database and start Ops Manager.

## (Optionally) Create a MongoDB object referencing the new Ops Manager

Now you can use the Ops Manager application to create MongoDB objects. You need to follow the 
[instructions](https://docs.mongodb.com/kubernetes-operator/stable/tutorial/install-k8s-operator/#onprem-prerequisites)
to prepare keys and enable network access to Ops Manager.

The Operator creates the NodePort service which can be used to access Ops Manager from external
(and therefore access the UI from browser):

```bash
$ kubectl get svc | grep <om-name>-svc-external
om-svc-external      NodePort    100.61.72.82    <none>        8080:30456/TCP    2m49s
```
Make sure that the firewall rules allow inbound traffic to the port on the host (`30456` in the example above)
 

## Deleting the Operator ##

It's important to keep correct order or removal operations. The simple rule is: **never remove Operator before mongodb resources**!
The reason is that the Operator cleans state in Ops Manager on deletion of the MongoDB resource in Kubernetes.

These are the correct steps to remove any MongoDB Operator resources:

```bash
# these three operations must be called first!
kubectl delete mdb --all -n <namespace>

# any of the following commands must be called after removing all existing mongodb resources
kubectl delete namespace <namespace>
kubectl delete deployment mongodb-enterprise-operator -n <namespace>
kubectl delete crd/mongodb.mongodb.com
```
