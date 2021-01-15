# MongoDB Enterprise Kubernetes Operator #

Welcome to the MongoDB Enterprise Kubernetes Operator. The Operator enables easy deploy of the following applications into Kubernetes clusters:
* MongoDB - Replica Sets, Sharded Clusters and Standalones, with authentication, TLS and many more options.
* Ops Manager - our enterprise management, monitoring and backup platform for MongoDB. The Operator can install and manage Ops Manager in Kubernetes for you. Ops Manager can manage MongoDB instances both inside and outside Kubernetes.

The Operator requires access to one of our database management tools - Ops Manager or Cloud Manager - to deploy MongoDB instances.
You may run Ops Manager either inside or outside Kubernetes, or may use Cloud Manager (cloud.mongodb.com) instead.

The Operator is currently Generally Available, supported by the [MongoDB Support Team](https://support.mongodb.com/). If you need urgent help, please file a support ticket.
For non-urgent requests, you may file a Github Issue here in the repo.

You can discuss this integration in our new [Community Forum](https://developer.mongodb.com/community/forums/) - please use the tag [kubernetes-operator](https://developer.mongodb.com/community/forums/tag/kubernetes-operator)

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

> In order to install the Operator in OpenShift, please follow [these](openshift-install.md) instructions instead.

This operator can also be installed using yaml files, in case you are not using Helm. You may apply the config directly from github clone this repo, and apply the file

    kubectl apply -f https://raw.githubusercontent.com/mongodb/mongodb-enterprise-kubernetes/master/mongodb-enterprise.yaml

or clone this repo, make any edits you need, and apply it from your machine.

    kubectl apply -f mongodb-enterprise.yaml

### Installation using Helm Chart

If you have installed the Helm client locally then you can run:

    helm install mongodb-enterprise-operator helm_chart --values helm_chart/values.yaml

You can customize installation by simple overriding of helm variables, for example use `--set operator.env="dev"` to run the Operator in development mode
(this will turn logging level to `Debug` and will make logging output as non-json)

Pass the `--values helm_chart/values-openshift.yaml` parameter if you want to install the Operator to an OpenShift cluster.
You need to specify the image pull secret name using `--set registry.imagePullSecrets=<secret_name>`

Check the end of the page for instructions on how to remove the Operator.

## MongoDB Resource ##

*This section describes how to create the MongoDB resource. Follow the next section on how to work with Ops Manager resource.*

### Adding Ops Manager Credentials ###

For the Operator to work, you will need the following information:

* Base URL - the URL of an Ops Manager instance (for Cloud Manager use `https://cloud.mongodb.com`)
* (optionally) Project Name - the name of an Ops Manager Project where MongoDBs will be deployed into. It will be
created by the Operator if it doesn't exist (and this is the recommended way instead of reusing the project created
in OpsManager directly). If omitted the name of the MongoDB resource will be used as a project name.
* (optionally) Organization ID - the ID of the organization which the Project belongs to. The Operator will create
an Organization with the same name as the Project if Organization ID is omitted.
* API Credentials. This can be any pair of:
  * Public and Private Programmatic API keys. They correspond to `user` and `publicApiKey` fields in the Secret storing
credentials. More information about the way to create them using Ops Manager UI can be found
[here](https://docs.opsmanager.mongodb.com/current/tutorial/configure-public-api-access/#programmatic-api-keys)
  * Username and Public API key. More information about the way to create them using Ops Manager UI can be found
 [here](https://docs.opsmanager.mongodb.com/current/tutorial/configure-public-api-access/#personal-api-keys-deprecated)

Note: that you must whitelist the Pod IP range of your Kubernetes cluster so that the Operator could make API requests to Ops Manager.
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
> Note, that if `orgId` is skipped then the new organization named `projectName` will be automatically created and new
project will be added there.
> If `projectName` is skipped the project created in Ops Manager will get the same name as the MongoDB object

Apply this file to create the new `Project`:

    kubectl apply -f my-project.yaml

### Credentials ###

For a user to be able to create or update objects in this Ops Manager Project they need either a Public API Key or a
Programmatic API Key. These will be held by Kubernetes as a `Secret` object. You can create this Secret with the following command:

``` bash
$ kubectl -n mongodb create secret generic my-credentials --from-literal="user=my-public-api-key" --from-literal="publicApiKey=my-private-api-key"
```

### Creating a MongoDB Resource ###

A MongoDB resource in Kubernetes is a MongoDB (short name `mdb`). We are going to create a replica set to test that everything is working as expected. There is a MongoDB replica set yaml file in `samples/mongodb/minimal/replica-set.yaml`.

If you have a correctly created Project with the name `my-project` and Credentials stored in a secret called `my-credentials` then, after applying this file then everything should be running and a new Replica Set with 3 members should soon appear in Ops Manager UI.

    kubectl apply -f samples/mongodb/minimal/replica-set.yaml -n mongodb

## MongoDBOpsManager Resource ##

This section describes how to create the Ops Manager Custom Resource in Kubernetes. Note, that this requires all
the CRDs and the Operator application to be installed as described above.

### Create Admin Credentials Secret ###

Before creating the Ops Manager resource you need to prepare the information about the admin user which will be
created automatically in Ops Manager. You can use the following command to do it:

```bash
$ kubectl create secret generic ops-manager-admin-secret  --from-literal=Username="jane.doe@example.com" --from-literal=Password="Passw0rd."  --from-literal=FirstName="Jane" --from-literal=LastName="Doe" -n <namespace>
```

Note, that the secret is needed only during the initialization of the Ops Manager object - you can remove it or
change the password using Ops Manager UI after the Ops Manager object was created

### Create MongoDBOpsManager Resource ###

Use the file `samples/ops-manager/ops-manager.yaml`. Edit the fields and create the object in Kubernetes:

```bash
$ kubectl apply -f samples/ops-manager/ops-manager.yaml -n <namespace>
```

Note, that it takes up to 8 minutes to initialize the Application Database and start Ops Manager.

### (Optionally) Create a MongoDB Resource Referencing the MongoDBOpsManager

Now you can use the Ops Manager application to create MongoDB objects. You need to follow the
[instructions](https://docs.mongodb.com/kubernetes-operator/stable/tutorial/create-operator-credentials/#prerequisites)
to prepare keys and enable network access to Ops Manager.
Then you need to perform the standard steps necessary to create MongoDB resource:
* Create a [credentials Secret](https://docs.mongodb.com/kubernetes-operator/stable/tutorial/create-operator-credentials/#create-k8s-credentials)
* Create a [connection ConfigMap](https://docs.mongodb.com/kubernetes-operator/stable/tutorial/create-project-using-configmap/)
  * Note, that you should use the value from `status.opsManager.url` in MongoDBOpsManager Resource as a value for `baseUrl` field in the ConfigMap

## Accessing Ops Manager UI (from a browser)

In order to access Ops Manager UI, from outside the Kubernetes cluster (from a browser), make sure you enable
`spec.externalConnectivity` in the Ops Manager resource definition. The easiest way is by configuring the LoadBalancer service type.

You will be able to fetch the URL to connect to Ops Manager UI from the `Service` created by the Operator.

## Deleting the Operator ##

It's important to keep correct order or removal operations. The simple rule is: **never remove Operator before MongoDB resources**!
The reason is that the Operator cleans state in Ops Manager on deletion of the MongoDB resource in Kubernetes.

These are the correct steps to remove any MongoDB Operator resources:

```bash
# this operation must be called first!
kubectl delete mdb --all -n <namespace>

# any of the following commands must be called after removing all existing mongodb resources
kubectl delete namespace <namespace>
kubectl delete deployment mongodb-enterprise-operator -n <namespace>
kubectl delete crd/mongodb.mongodb.com
kubectl delete crd/opsmanagers.mongodb.com
kubectl delete crd/mongodbusers.mongodb.com
```

## Contributing

Please file issues before filing PRs. For PRs to be accepted, contributors must sign our [CLA](https://www.mongodb.com/legal/contributor-agreement).

Reviewers, please ensure that the CLA has been signed by referring to [the contributors tool](https://contributors.corp.mongodb.com/) (internal link).
