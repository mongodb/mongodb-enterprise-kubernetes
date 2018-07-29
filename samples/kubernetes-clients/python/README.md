The following examples using the [Official Python client library for Kubernetes](https://github.com/kubernetes-client/python) show how to:

- Deploy the Kubernetes Operator including the required 
   - [ClusterRole](https://kubernetes.io/docs/reference/access-authn-authz/rbac/#default-roles-and-role-bindings), 
   - [ClusterRoleBinding](https://kubernetes.io/docs/reference/access-authn-authz/rbac/#default-roles-and-role-bindings) and 
   - [ServiceAccount](https://kubernetes.io/docs/reference/access-authn-authz/service-accounts-admin/)
   
- Creation and deletion of MongoDB the following type of deployments:
   - Standalone
   - Replica Set
   - Sharded Cluster

For more details, please refer to the repository for the Python client library for Kubernetes: https://github.com/kubernetes-client/python


**NOTE**: the given examples assume the existence of the Kubernetes namespace `mongodb`. If using a different namespace, please modify the relevant variable in the sample code.