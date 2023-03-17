# Multi-Cluster CLI GitOps Samples

This is an example of using the `multi-cluster-cli` in a [GitOps](https://www.weave.works/technologies/gitops/) operating model to perform a recovery of the dataplane
in a multi-cluster deployment scenario. For more details on managing multi-cluster resources with the kubernetes operator see [the official documentation](https://www.mongodb.com/docs/kubernetes-operator/master/multi-cluster/). The example is applicable for an [ArgoCD](https://argo-cd.readthedocs.io/) configuration.

## ArgoCD configuration
The files in the [argocd](./argocd) contain an [AppProject](./argocd/project.yaml) and an [Application](./argocd/application.yaml) linked to it which allows the synchronization of `MongoDBMulti` resources from a Git repo.

## Multi-Cluster CLI Job setup
To enable the manual disaster recovery using the CLI, this sample provides a [Job](./resources/job.yaml) which runs the recovery subcommand as a [PreSync hook](https://argo-cd.readthedocs.io/en/stable/user-guide/resource_hooks/). This ensures that the multicluster environment is configured before the application of the modified [`MongoDBMulti`](./resources/replica-set.yaml) resource. The `Job` mounts the same `kubeconfig` that the operator is using to connect to the clusters defined in your architecture. 

## RBAC Settings for the Central and Member clusters
The RBAC settings for the operator are typically creating using the CLI. In cases, where it is not possible, you can adjust and apply the YAML files from the [rbac](./resources/rbac) directory.

### Build the multi-cluster CLI image
You can build a minimal image containing the CLI executable using the `Dockerfile` [provided in this repo](./../../tools/multicluster/Dockerfile).
``` shell
git clone https://github.com/mongodb/mongodb-enterprise-kubernetes
cd mongodb-enterprise-kubernetes/tools/multicluster
docker build . -t "your-registry/multi-cluster-cli:latest"
docker push "your-registry/multi-cluster-cli:latest"
```
