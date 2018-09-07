The following examples using the [Official Python client library for Kubernetes](https://github.com/kubernetes-client/python) show how to:

- Creation of the following Kubernetes objects:
   - Config map for the Ops/Cloud Manager project
   - Secret for the API user
   
- Create and delete the following type of MongoDB deployments:
   - Standalone
   - Replica Set
   - Sharded Cluster

The sample code has been tested with Python 2.7 and 3.6. 

For more details, please refer to the repository for the Python client library for Kubernetes: https://github.com/kubernetes-client/python

**NOTE**: the given example assume the existence of the following:
 - namespace `mongodb`
 - ClusterRole/Role `mongodb-enterprise-operator`
 - ClusterRoleBinding/RoleBinding `mongodb-enterprise-operator` 
 - ServiceAccount `mongodb-enterprise-operator`
 
 If using a different namespace, please modify the relevant variable in the sample code.