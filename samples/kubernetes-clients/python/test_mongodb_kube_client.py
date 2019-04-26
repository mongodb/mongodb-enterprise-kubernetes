#!/usr/bin/env python
from __future__ import print_function

import yaml

from mongodb_kube_client import MongoDBEnterpriseKubeClient


def parse_config_file(path):
    '''
    Parses the config file in the given path
    '''
    with open(path, 'r') as parameters:
        try:
            return yaml.load(parameters)
        except yaml.YAMLError as exc:
            print("Error when loading environment variables", exc)


def main():
    parameters = parse_config_file("mongodb_kube_operator.cfg")

    namespace = parameters["kubernetes"]["namespace"]
    om_project = parameters["ops_manager"]["project"]
    om_base_url = parameters["ops_manager"]["base_url"]
    om_api_user = parameters["ops_manager"]["api_user"]
    om_api_key = parameters["ops_manager"]["api_key"]

    # Instantiate client wrapper
    kube_client = MongoDBEnterpriseKubeClient(namespace, om_api_user, om_api_key, om_project, om_base_url)

    # Create a secret and config map for project
    kube_client.create_secret()
    kube_client.create_config_map()

    # Create a standalone, replica set and sharded cluster

    kube_client.deploy_standalone(mongo_version="4.0.0", name="my-standalone")

    kube_client.deploy_replica_set(mongo_version="4.0.0", name="my-replica-set", members=3)

    kube_client.deploy_sharded_cluster(mongo_version="4.0.0", name="my-sharded-cluster",
                                       num_mongod_per_shard=3, num_shards=2,
                                       num_cfg_rs_members=3, num_mongos=2)
    '''
    # Delete the created deployments

    kube_client.delete_mongo_process(name="my-standalone", type_plural="mongodb")

    kube_client.delete_mongo_process(name="my-replica-set", type_plural="mongodb")

    kube_client.delete_mongo_process(name="my-sharded-cluster", type_plural="mongodb")
    '''


if __name__ == '__main__':
    main()
