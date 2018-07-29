from __future__ import print_function

import base64
import os
from pprint import pprint

from kubernetes import client, config
from kubernetes.client.rest import ApiException

'''
Sample class to interact with Kubernetes API and MongoDB Kubernetes Operator
'''
class KubeClient(object):

    def __init__(self):

        # Get local configuration from HOME
        home_folder = os.path.expanduser('~')
        config.load_kube_config(os.path.join(home_folder, '.kube/config'))

        # Instanciate Core V1 API
        self.v1 = client.CoreV1Api()

        # Instanciate RBAC Auth V1 API
        self.rbac_auth_v1 = client.RbacAuthorizationV1Api()

        # Instanciate Apps V1 API
        self.apps_v1 = client.AppsV1Api()

        # Instanciate Custom Objects API - for creating MongoDB deployments
        self.custom_obj = client.CustomObjectsApi()

        # Namespace; must be already created in the Kubernetes Cluster
        self.namespace = "mongodb"

        # Ops Manager API information
        self.om_api_user = "first.last@example.com"
        self.om_api_key = "my-public-api-key"

        # Ops Manager project information
        self.om_project_id = "my-project-id"
        self.om_base_url = "https://my-ops-cloud-manager-url"

    '''
    Equivalent to .yaml file:
    ---
    kind: ClusterRole
    apiVersion: rbac.authorization.k8s.io/v1
    metadata:
      name: mongodb-enterprise-operator
    rules:
    - apiGroups:
      - ""
      resources:
      - configmaps
      - secrets
      - services
      verbs:
      - get
      - list
      - create
      - update
      - delete
    - apiGroups:
      - apps
      resources:
      - statefulsets
      verbs: ["*"]
    - apiGroups:
      - apiextensions.k8s.io
      resources:
      - customresourcedefinitions
      verbs:
      - get
      - list
      - watch
      - create
      - delete
    - apiGroups:
      - mongodb.com
      resources:
      - "*"
      verbs:
      - "*"
    '''
    def __create_cluster_role(self):

        print("Creating clusterrole named mongodb-enterprise-operator")

        rules = [client.V1PolicyRule(api_groups=[""], resources=["configmaps", "secrets", "services"],
                                     verbs=["get", "list", "create", "update", "delete"]),
                 client.V1PolicyRule(api_groups=["apps"], resources=["statefulsets"], verbs=["*"]),
                 client.V1PolicyRule(api_groups=["apiextensions.k8s.io"], resources=["customresourcedefinitions"],
                                     verbs=["get", "list", "watch", "create", "delete"]),
                 client.V1PolicyRule(api_groups=["mongodb.com"], resources=["*"], verbs=["*"])]

        body = client.V1ClusterRole(
            api_version="rbac.authorization.k8s.io/v1",
            kind="ClusterRole",
            metadata=client.V1ObjectMeta(name="mongodb-enterprise-operator"),
            rules=rules)

        try:
            api_response = self.rbac_auth_v1.create_cluster_role(body)
            pprint(api_response)
        except ApiException as e:
            print("Exception when creating required Cluster Role: %s\n" % e)


    '''
    Equivalent to .yaml file:
    ---
    apiVersion: v1
    kind: ServiceAccount
    metadata:
      name: mongodb-enterprise-operator
      namespace: mongodb
    '''
    def __create_service_account(self):

        print("Creating service account named mongodb-enterprise-operator")

        metadata = client.V1ObjectMeta(name="mongodb-enterprise-operator", namespace=self.namespace)

        body = client.V1ServiceAccount(api_version="v1", kind="ServiceAccount", metadata=metadata)

        try:
            api_response = self.v1.create_namespaced_service_account(self.namespace, body)
            pprint(api_response)
        except ApiException as e:
            print("Exception when creating Service Account: %s\n" % e)

    '''
    ---
    kind: ClusterRoleBinding
    apiVersion: rbac.authorization.k8s.io/v1
    metadata:
      name: mongodb-enterprise-operator
      namespace: mongodb
    roleRef:
      apiGroup: rbac.authorization.k8s.io
      kind: ClusterRole
      name: mongodb-enterprise-operator
    subjects:
    - kind: ServiceAccount
      name: mongodb-enterprise-operator
      namespace: mongodb
    '''
    def __create_cluster_role_binding(self):

        print("Creating cluster role binding named mongodb-enterprise-operator")

        metadata = client.V1ObjectMeta(name="mongodb-enterprise-operator", namespace=self.namespace)

        subjects = [client.V1Subject(kind="ServiceAccount", name="mongodb-enterprise-operator",
                                     namespace=self.namespace)]

        role_reference = client.V1RoleRef(api_group="rbac.authorization.k8s.io", kind="ClusterRole",
                                          name="mongodb-enterprise-operator")

        body = client.V1ClusterRoleBinding(api_version="rbac.authorization.k8s.io/v1",
                                           metadata=metadata,
                                           role_ref=role_reference,
                                           subjects=subjects)

        try:
            api_response = self.rbac_auth_v1.create_cluster_role_binding(body)
            pprint(api_response)
        except ApiException as e:
            print("Exception when creating Cluster Role Binding: %s\n" % e)

    '''
    ---
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: mongodb-enterprise-operator
      namespace: mongodb
    spec:
      replicas: 1
      selector:
        matchLabels:
          app: mongodb-enterprise-operator
      template:
        metadata:
          labels:
            app: mongodb-enterprise-operator
        spec:
          serviceAccountName: mongodb-enterprise-operator
          containers:
          - name: mongodb-enterprise-operator
            image: quay.io/mongodb/mongodb-enterprise-operator:latest
            imagePullPolicy: Always
            env:
            - name: OPERATOR_ENV
              value: "local"
            - name: MONGODB_ENTERPRISE_DATABASE_IMAGE
              value: quay.io/mongodb/mongodb-enterprise-database:latest
            - name: IMAGE_PULL_POLICY
              value: Always
            - name: IMAGE_PULL_SECRETS
              value: ""
    '''
    def deploy_operator(self):

        # Creating requirements for deploying the Operator
        self.__create_cluster_role()
        self.__create_service_account()
        self.__create_cluster_role_binding()

        # Operator default Deployment specification
        metadata = client.V1ObjectMeta(name="mongodb-enterprise-operator",
                                       namespace=self.namespace)

        env_variables = [client.V1EnvVar(name="OPERATOR_ENV", value="local"),
                         client.V1EnvVar(name="MONGODB_ENTERPRISE_DATABASE_IMAGE",
                                         value="quay.io/mongodb/mongodb-enterprise-database:latest"),
                         client.V1EnvVar(name="IMAGE_PULL_POLICY", value="Always"),
                         client.V1EnvVar(name="IMAGE_PULL_SECRETS", value="")]

        containers = [client.V1Container(name="mongodb-enterprise-operator",
                                         image="quay.io/mongodb/mongodb-enterprise-operator:latest",
                                         image_pull_policy="Always", env=env_variables)]

        pod_spec = client.V1PodSpec(service_account_name="mongodb-enterprise-operator",
                                    containers=containers)

        template = client.V1PodTemplateSpec(metadata=client.V1ObjectMeta(labels={"app": "mongodb-enterprise-operator"}),
                                            spec=pod_spec)

        spec = client.V1DeploymentSpec(replicas=1, selector=client.V1LabelSelector(
            match_labels={"app": "mongodb-enterprise-operator"}),
                                       template=template)

        body = client.V1Deployment(api_version="apps/v1", kind="Deployment", spec=spec, metadata=metadata)

        try:
            api_response = self.apps_v1.create_namespaced_deployment(self.namespace, body)
            pprint(api_response)
        except ApiException as e:
            print("Exception when creating the Ops Manager Kubernetes Operator: %s\n" % e)

    '''
    Create secret: https://docs.opsmanager.mongodb.com/current/tutorial/install-k8s-operator/index.html#create-credentials

    Equivalent to execute:
    kubectl -n mongodb create secret generic \
      my-credentials --from-literal="user=<first.last@example.com>" \
      --from-literal="publicApiKey=<my-public-api-key>"
    '''

    def create_secret(self):
        print("Creating secret for user named %s with the provided API key" % self.om_api_user)
        metadata = client.V1ObjectMeta(name="my-credentials", namespace=self.namespace)

        # Encode credentials
        try: #Python 3.6
            encoded_user = base64.encodebytes(bytes(self.om_api_user, "utf-8"))
            encoded_key = base64.encodebytes(bytes(self.om_api_key, "utf-8"))
        except AttributeError: #Python 2.7
            encoded_user = base64.b64encode(self.om_api_user)
            encoded_key = base64.b64encode(self.om_api_key)

        # Transform binary into string
        encoded_user = encoded_user.decode("utf-8").rstrip("\n")
        encoded_key = encoded_key.decode("utf-8").rstrip("\n")

        body = client.V1Secret(api_version="v1", kind="Secret", metadata=metadata, type="from-literal",
                               data={"user": encoded_user,
                                     "publicApiKey": encoded_key})

        try:
            api_response = self.v1.create_namespaced_secret(self.namespace, body)
            pprint(api_response)
        except ApiException as e:
            print("Exception when creating secret: %s\n" % e)

    '''
    Create a config map: https://docs.opsmanager.mongodb.com/current/tutorial/install-k8s-operator/index.html#create-onprem-project

    Equivalent .yaml file:
    ---
    apiVersion: v1
    kind: ConfigMap
    metadata:
      name: my-project
      namespace: mongodb
    data:
      projectId: my-project-id
      baseUrl: https://my-ops-cloud-manager-url
    '''

    def create_config_map(self):
        print("Creating config map for project ID %s with base URL %s" % (self.om_project_id, self.om_base_url))
        metadata = client.V1ObjectMeta(name="my-project", namespace=self.namespace)

        body = client.V1ConfigMap(api_version="v1", kind="ConfigMap", metadata=metadata,
                                  data={"projectId": self.om_project_id,
                                        "baseUrl": self.om_base_url})

        try:
            api_response = self.v1.create_namespaced_config_map(self.namespace, body)
            self.__print_api_response(api_response)
        except ApiException as e:
            print("Exception when creating config map: %s\n" % e)

    '''
    Creating a standalone MongoDB process.

    Equivalent to .yaml file:
    ---
    apiVersion: mongodb.com/v1
    kind: MongoDbStandalone
    metadata:
      name: <name_param>
      namespace: mongodb
    spec:
      version: <mongo_version_param>

      project: my-project
      credentials: my-credentials

      persistent: true

    '''

    def deploy_standalone(self, mongo_version, name):
        group = 'mongodb.com'
        version = 'v1'
        plural = 'mongodbstandalones'

        body = {"spec":
                    {"persistent": True, "version": str(mongo_version), "credentials": "my-credentials", "project": "my-project"},
                "kind": "MongoDbStandalone", "apiVersion": "mongodb.com/v1",
                "metadata": {"name": name, "namespace": self.namespace}}

        try:
            api_response = self.custom_obj.create_namespaced_custom_object(group, version, self.namespace, plural, body)
            pprint(api_response)
        except ApiException as e:
            print("Exception when creating a MongoDB standalone process: %s\n" % e)


    '''
    Creating a MongoDB Replica Set.

    Equivalent to .yaml file:
    ---
    apiVersion: mongodb.com/v1
    kind: MongoDbReplicaSet
    metadata:
      name: <name_param>
      namespace: mongodb
    spec:
      members: <members_param>
      version: <mongo_version_param>

      project: my-project
      credentials: my-credentials

      persistent: true
    '''

    def deploy_replica_set(self, mongo_version, name, members=3):
        group = 'mongodb.com'
        version = 'v1'
        plural = 'mongodbreplicasets'

        body = {"spec": {"members": members, "persistent": True, "version": str(mongo_version),
                         "credentials": "my-credentials",
                         "project": "my-project"},
                "kind": "MongoDbReplicaSet", "apiVersion": "mongodb.com/v1",
                "metadata": {"name": name, "namespace": self.namespace}}

        try:

            api_response = self.custom_obj.create_namespaced_custom_object(group, version, self.namespace, plural, body)
            pprint(api_response)
        except ApiException as e:
            print("Exception when creating a MongoDB Replica Set: %s\n" % e)

    '''
    Creating a MongoDB Sharded Cluster.

    Equivalent to .yaml file:
    ---
    apiVersion: mongodb.com/v1
    kind: MongoDbShardedCluster
    metadata:
      name: <name_param>
      namespace: mongodb
    spec:
      shardCount: <num_shards_param>
      mongodsPerShardCount: <num_mongod_per_shard_param>
      mongosCount: <num_mongos_param>
      configServerCount: <num_cfg_rs_members_param>
      version: <mongo_version_param>

      project: my-project
      credentials: my-credentials

      persistent: true
    '''

    def deploy_sharded_cluster(self, mongo_version, name, num_shards, num_mongos, num_mongod_per_shard=3,
                               num_cfg_rs_members=3):

        group = 'mongodb.com'
        version = 'v1'
        plural = 'mongodbshardedclusters'

        body = {"spec": {"shardCount": num_shards, "mongodsPerShardCount": num_mongod_per_shard,
                         "mongosCount": num_mongos, "persistent": False, "version": mongo_version,
                         "configServerCount": num_cfg_rs_members,
                         "credentials": "my-credentials",
                         "project": "my-project"},
                "kind": "MongoDbShardedCluster", "apiVersion": "mongodb.com/v1",
                "metadata": {"name": name, "namespace": self.namespace}}

        try:
            api_response = self.custom_obj.create_namespaced_custom_object(group, version, self.namespace, plural, body)
            pprint(api_response)
        except ApiException as e:
            print("Exception when creating a MongoDB Sharded Cluster: %s\n" % e)


    '''
    Delete MongoDB deployments by name and type

    type_plural can be: "mongodbstandalones", "mongodbreplicasets" or "mongodbshardedclusters"
    '''

    def delete_mongo_process(self, name, type_plural):
        group = 'mongodb.com'
        version = 'v1'
        namespace = self.namespace
        plural = type_plural
        body = client.V1DeleteOptions(propagation_policy="Background")
        grace_period_seconds = 56
        orphan_dependents = False

        try:
            api_response = self.custom_obj.delete_namespaced_custom_object(group, version, namespace, plural, name, body,
                                                                        grace_period_seconds=grace_period_seconds,
                                                                        orphan_dependents=orphan_dependents)
            pprint(api_response)
            if api_response["status"] == "Success":
                return True
        except ApiException as e:
            print("Exception when deleting MongoDB deployment: %s\n" % e)
        return False

#Instanciate client wrapper
kube_client = KubeClient()

#Deploy the Kubernetes Operator
kube_client.deploy_operator()

#Create a secret and config map for project
kube_client.create_secret()
kube_client.create_config_map()

# Create a standalone, replica set and sharded cluster
kube_client.deploy_standalone(mongo_version="4.0.0", name="my-standalone")

kube_client.deploy_replica_set(mongo_version="4.0.0", name="my-replica-set", members=3)

kube_client.deploy_sharded_cluster(mongo_version="4.0.0", name="my-sharded-cluster",
                                   num_mongod_per_shard=3, num_shards=2,
                                   num_cfg_rs_members=3, num_mongos=2)

#Delete the created deployments
kube_client.delete_mongo_process(name="my-standalone", type_plural="mongodbstandalones")

kube_client.delete_mongo_process(name="my-replica-set", type_plural="mongodbreplicasets")

kube_client.delete_mongo_process(name="my-sharded-cluster", type_plural="mongodbshardedclusters")