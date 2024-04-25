# MongoDB Enterprise Kubernetes Operator #

Welcome to the MongoDB Enterprise Kubernetes Operator. The Operator enables easy deploy of the following applications into Kubernetes clusters:
* MongoDB - Replica Sets, Sharded Clusters and Standalones - with authentication, TLS and many more options.
* Ops Manager - our enterprise management, monitoring and backup platform for MongoDB. The Operator can install and manage Ops Manager in Kubernetes for you. Ops Manager can manage MongoDB instances both inside and outside Kubernetes.

The Operator requires access to one of our database management tools - Ops Manager or Cloud Manager - to deploy MongoDB instances. You may run Ops Manager either inside or outside Kubernetes, or may use Cloud Manager (cloud.mongodb.com) instead.

This is an Enterprise product, available under the Enterprise Advanced license.
We also have a [Community Operator](https://github.com/mongodb/mongodb-kubernetes-operator).

## Support, Feature Requests and Community ##

The Enterprise Operator is supported by the [MongoDB Support Team](https://support.mongodb.com/). If you need help, please file a support ticket.
If you have a feature request, you can make one on our [Feedback Site](https://feedback.mongodb.com/forums/924355-ops-tools)

You can discuss this integration in our new [Community Forum](https://developer.mongodb.com/community/forums/) - please use the tag [kubernetes-operator](https://developer.mongodb.com/community/forums/tag/kubernetes-operator)

## Videos ##

Here are some talks from MongoDB Live 2020 about the Operator:
* [Kubernetes, MongoDB, and Your MongoDB Data Platform](https://www.youtube.com/watch?v=o1fUPIOdKeU)
* [Run it in Kubernetes! Community and Enterprise MongoDB in Containers](https://www.youtube.com/watch?v=2Xszdg-4T6A)

## Documentation ##

[Install Kubernetes Operator](https://docs.opsmanager.mongodb.com/current/tutorial/install-k8s-operator)

[Deploy MongoDB](https://docs.mongodb.com/kubernetes-operator/stable/mdb-resources/)

[Deploy Ops Manager](https://docs.mongodb.com/kubernetes-operator/stable/om-resources/)

[MongoDB Resource Specification](https://docs.opsmanager.mongodb.com/current/reference/k8s-operator-specification)

[Ops Manager Resource Specification](https://docs.mongodb.com/kubernetes-operator/stable/reference/k8s-operator-om-specification/)

[Troubleshooting Kubernetes Operator](https://docs.opsmanager.mongodb.com/current/reference/troubleshooting/k8s/)

[Known Issues for Kubernetes Operator](https://docs.mongodb.com/kubernetes-operator/stable/reference/known-issues/)

## Requirements ##

Please refer to the [Installation Instructions](https://docs.mongodb.com/kubernetes-operator/stable/tutorial/plan-k8s-operator-install/)
to see which Kubernetes and Openshift versions the Operator is compatible with

To work with MongoDB resource this Operator requires [Ops Manager](https://docs.opsmanager.mongodb.com/current/) (Ops Manager can
be installed into the same Kubernetes cluster by the Operator or installed outside of the cluster manually)
or [Cloud Manager](https://cloud.mongodb.com/user#/cloud/login).
> If this is your first time trying the Operator, Cloud Manager is easier to get started. Log in, and create 'Cloud Manager' Organizations and Projects to use with the Operator.


## Installation

### Create Kubernetes Namespace

The Mongodb Enterprise Operator is installed, into the `mongodb` namespace by default, but this namespace is not created automatically. To create this namespace you should execute:

    kubectl create namespace mongodb

To use a different namespace, update the yaml files' `metadata.namespace` attribute to point to your preferred namespace.  If using `helm` you need to override the `namespace` attribute with `--set namespace=<..>` during helm installation.

### Installation using yaml files

#### Create CustomResourceDefinitions

`CustomResourceDefinition`s (or `CRDs`) are Kubernetes Objects which can be used to instruct the Operators to perform operations on your Kubernetes cluster. Our CRDs control MongoDB and Ops Manager deployments. They should be installed before installing the Operator.
CRDs are defined cluster-wide, so to install them, you must have Cluster-level access. However, once the CRDs are installed, MongoDB instances can be deployed with namespace-level access only.

    kubectl apply -f https://raw.githubusercontent.com/mongodb/mongodb-enterprise-kubernetes/master/crds.yaml

#### Operator Installation

> In order to install the Operator in OpenShift, please follow [these](openshift-install.md) instructions instead.

To install the Operator using yaml files, you may apply the config directly from github;

    kubectl apply -f https://raw.githubusercontent.com/mongodb/mongodb-enterprise-kubernetes/master/mongodb-enterprise.yaml

or can clone this repo, make any edits you need, and apply it from disk:

    kubectl apply -f mongodb-enterprise.yaml

### Installation using the Helm Chart

MongoDB's official Helm Charts are hosted at https://github.com/mongodb/helm-charts

## MongoDB Resource ##

*This section describes how to deploy MongoDB instances. This requires a working Ops or Cloud Manager installation. See below for instructions on how to configure Ops Manager.*

### Adding Ops Manager Credentials ###

For the Operator to work, you will need the following information:

* Base URL - the URL of an Ops Manager instance (for Cloud Manager use `https://cloud.mongodb.com`)
* (optional) Project Name - the name of an Ops Manager Project for MongoDB instances to be deployed into. This project will be created by the Operator if it doesn't exist. We recommend that you allow the Operator to create and manage the projects it uses. By default, the Operator will use the name of the MongoDB resource as the project name.
* (optional) Organization ID - the ID of the Organization which the Project belongs to. By default, the Operator will create an Organization with the same name as the Project.
* API Credentials. This can be any pair of:
  * Public and Private Programmatic API keys. They correspond to `user` and `publicApiKey` fields in the Secret storing
credentials. More information about the way to create them using Ops Manager UI can be found
[here](https://docs.opsmanager.mongodb.com/current/tutorial/configure-public-api-access/#programmatic-api-keys)
  * Username and Public API key. More information about the way to create them using Ops Manager UI can be found
 [here](https://docs.opsmanager.mongodb.com/current/tutorial/configure-public-api-access/#personal-api-keys-deprecated)

Note: When creating API credentials, you must allow the Pod IP range of your Kubernetes cluster to use the credentials - otherwise, API requests from the Operator to Ops Manager will be rejected.
You can get the Pod IP range of your kubernetes cluster by executing the command: ```kubectl cluster-info dump | grep -m 1 cluster-cidr```

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
  projectName: myProjectName # this is an optional parameter
  orgId: 5b890e0feacf0b76ff3e7183 # this is an optional parameter
  baseUrl: https://my-ops-manager-or-cloud-manager-url
```
> `projectName` is optional, and the value of `metadata.name` will be used if it is not defined.
> `orgId` is required.

Apply this file to create the new `Project`:

    kubectl apply -f my-project.yaml

### Credentials ###

For a user to be able to create or update objects in this Ops Manager Project they need either a Public API Key or a
Programmatic API Key. These will be held by Kubernetes as a `Secret` object. You can create this Secret with the following command:

``` bash
$ kubectl -n mongodb create secret generic my-credentials --from-literal="user=my-public-api-key" --from-literal="publicApiKey=my-private-api-key"
```

### Creating a MongoDB Resource ###

A MongoDB resource in Kubernetes is a MongoDB. We are going to create a replica set to test that everything is working as expected. There is a MongoDB replica set yaml file in `samples/mongodb/minimal/replica-set.yaml`.

If you have a Project with the name `my-project` and Credentials stored in a secret called `my-credentials`, then after applying this file everything should be running and a new Replica Set with 3 members should soon appear in Ops Manager UI.

    kubectl apply -f samples/mongodb/minimal/replica-set.yaml -n mongodb

## MongoDBOpsManager Resource ##

This section describes how to create the Ops Manager Custom Resource in Kubernetes. Note, that this requires all
the CRDs and the Operator application to be installed as described above.

### Create Admin Credentials Secret ###

Before creating the Ops Manager resource you need to prepare the information about the admin user which will be
created automatically in Ops Manager. You can use the following command to do it:

```bash
$ kubectl create secret generic ops-manager-admin-secret  --from-literal=Username="user.name@example.com" --from-literal=Password="Passw0rd."  --from-literal=FirstName="User" --from-literal=LastName="Name" -n <namespace>
```

Note, that the secret is needed only during the initialization of the Ops Manager object - you can remove it or
change the password using Ops Manager UI after the Ops Manager object is created.

### Create MongoDBOpsManager Resource ###

Use the file `samples/ops-manager/ops-manager.yaml`. Edit the fields and create the object in Kubernetes:

```bash
$ kubectl apply -f samples/ops-manager/ops-manager.yaml -n <namespace>
```

Note, that it can take up to 8 minutes to initialize the Application Database and start Ops Manager.

## Accessing the Ops Manager UI using your web browser

In order to access the Ops Manager UI from outside the Kubernetes cluster, you must enable `spec.externalConnectivity` in the Ops Manager resource definition. The easiest approach is by configuring the LoadBalancer service type.

You will be able to fetch the URL to connect to Ops Manager UI from the `Service` object created by the Operator.

## Removing the Operator, Databases and Ops Manager from your Kubernetes cluster ##

As the Operator manages MongoDB and Ops Manager resources, if you want to remove them from your Kubernetes cluster, database instances and Ops Manager must be removed before removing the Operator. Removing the Operator first, or deleting the namespace will cause delays or stall the removal process of MongoDB objects, requiring manual intervention.

Here is the correct order to completely remove the Operator and the services managed by it:

* Remove all database clusters managed by the Operator
* Remove Ops Manager
* Remove the Operator
* Remove the CRDs

## Contributing

For PRs to be accepted, all contributors must sign our [CLA](https://www.mongodb.com/legal/contributor-agreement).

Reviewers, please ensure that the CLA has been signed by referring to [the contributors tool](https://contributors.corp.mongodb.com/) (internal link).
