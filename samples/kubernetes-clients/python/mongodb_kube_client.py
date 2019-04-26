from __future__ import print_function

import base64

from pprint import pprint
from kubernetes import client, config
from kubernetes.client.rest import ApiException

# Added for Python2 and Python3 cross-compatibility
try:
    def base64encode(input_string):
        return base64.encodebytes(bytes(input_string, "utf-8"))
except:
    base64encode = base64.b64encode


class MongoDBEnterpriseKubeClient(object):

    def __init__(self, namespace, om_api_user, om_api_key, om_project_id, om_base_url):

        config.load_kube_config()

        # Instantiate Core V1 API
        self.v1 = client.CoreV1Api()

        # Instantiate RBAC Auth V1 API
        self.rbac_auth_v1 = client.RbacAuthorizationV1Api()

        # Instantiate Apps V1 API
        self.apps_v1 = client.AppsV1Api()

        # Instantiate Custom Objects API - for creating MongoDB deployments
        self.custom_obj = client.CustomObjectsApi()

        # Namespace; must be already created in the Kubernetes Cluster
        self.namespace = namespace

        # Ops Manager API information
        self.om_api_user = om_api_user
        self.om_api_key = om_api_key

        # Ops Manager project information
        self.om_project_id = om_project_id
        self.om_base_url = om_base_url

    def create_secret(self):
        '''
        Create secret:
        https://docs.opsmanager.mongodb.com/current/tutorial/install-k8s-operator/index.html#create-credentials

        Equivalent to execute:
        kubectl -n mongodb create secret generic \
          my-credentials --from-literal="user=<first.last@example.com>" \
          --from-literal="publicApiKey=<my-public-api-key>"
        '''

        print("Creating secret for user named %s with the provided API key" % self.om_api_user)
        metadata = client.V1ObjectMeta(name="my-credentials", namespace=self.namespace)

        # Encode credentials
        encoded_user = base64encode(self.om_api_user)
        encoded_key = base64encode(self.om_api_key)

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

    def create_config_map(self):
        '''
        Create a config map:
        https://docs.opsmanager.mongodb.com/current/tutorial/install-k8s-operator/index.html#create-onprem-project
        '''

        print("Creating config map for project ID %s with base URL %s" % (self.om_project_id, self.om_base_url))
        metadata = client.V1ObjectMeta(name="my-project", namespace=self.namespace)

        body = client.V1ConfigMap(api_version="v1", kind="ConfigMap", metadata=metadata,
                                  data={"projectId": self.om_project_id,
                                        "baseUrl": self.om_base_url})

        try:
            api_response = self.v1.create_namespaced_config_map(self.namespace, body)
            pprint(api_response)
        except ApiException as e:
            print("Exception when creating config map: %s\n" % e)

    def deploy_standalone(self, mongo_version, name):
        '''
        Creating a standalone MongoDB process.
        '''

        group = 'mongodb.com'
        version = 'v1'
        plural = 'mongodb'

        body = {"spec":
                    {"persistent": False, "version": str(mongo_version), "credentials": "my-credentials",
                     "project": "my-project"},
                "kind": "MongoDB", "apiVersion": "mongodb.com/v1",
                "metadata": {"name": name, "namespace": self.namespace}}

        try:
            api_response = self.custom_obj.create_namespaced_custom_object(group, version, self.namespace, plural, body)
            pprint(api_response)
        except ApiException as e:
            print("Exception when creating a MongoDB standalone process: %s\n" % e)

    def deploy_replica_set(self, mongo_version, name, members=3):
        '''
        Creating a MongoDB Replica Set.
        '''

        group = 'mongodb.com'
        version = 'v1'
        plural = 'mongodb'

        body = {"spec": {"members": members, "persistent": False, "version": str(mongo_version),
                         "credentials": "my-credentials",
                         "project": "my-project"},
                "kind": "MongoDB", "apiVersion": "mongodb.com/v1",
                "metadata": {"name": name, "namespace": self.namespace}}

        try:

            api_response = self.custom_obj.create_namespaced_custom_object(group, version, self.namespace, plural, body)
            pprint(api_response)
        except ApiException as e:
            print("Exception when creating a MongoDB Replica Set: %s\n" % e)

    def deploy_sharded_cluster(self, mongo_version, name, num_shards, num_mongos, num_mongod_per_shard=3,
                               num_cfg_rs_members=3):
        '''
        Creating a MongoDB Sharded Cluster.
        '''

        group = 'mongodb.com'
        version = 'v1'
        plural = 'mongodb'

        body = {"spec": {"shardCount": num_shards, "mongodsPerShardCount": num_mongod_per_shard,
                         "mongosCount": num_mongos, "persistent": False, "version": mongo_version,
                         "configServerCount": num_cfg_rs_members,
                         "credentials": "my-credentials",
                         "project": "my-project"},
                "kind": "MongoDB", "apiVersion": "mongodb.com/v1",
                "metadata": {"name": name, "namespace": self.namespace}}

        try:
            api_response = self.custom_obj.create_namespaced_custom_object(group, version, self.namespace, plural, body)
            pprint(api_response)
        except ApiException as e:
            print("Exception when creating a MongoDB Sharded Cluster: %s\n" % e)

    def delete_mongo_process(self, name, type_plural):
        '''
        Delete MongoDB deployments by name and type
        '''

        group = 'mongodb.com'
        version = 'v1'
        namespace = self.namespace
        plural = type_plural
        body = client.V1DeleteOptions(propagation_policy="Background")
        grace_period_seconds = 56
        orphan_dependents = False

        try:
            api_response = self.custom_obj.delete_namespaced_custom_object(group, version, namespace, plural, name,
                                                                           body,
                                                                           grace_period_seconds=grace_period_seconds,
                                                                           orphan_dependents=orphan_dependents)
            pprint(api_response)
            if api_response["status"] == "Success":
                return True
        except ApiException as e:
            print("Exception when deleting MongoDB deployment: %s\n" % e)
        return False


if __name__ == '__main__':
    pass
