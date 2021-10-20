
# Open Policy Agent Samples

This is a library of sample policies for [OPA Gatekeeper](https://open-policy-agent.github.io/gatekeeper/website/docs/) . You can edit and apply any of the policies or use them as a springboard to create your own. Each policy is enclosed in its own directory in the form of a policy_name.yaml file ([Constraint Template](https://open-policy-agent.github.io/gatekeeper/website/docs/constrainttemplates)) and constraints.yaml (Constraint).

**Instructions for use**

You will need OPA Gatekeeper installed on your Kubernetes cluster. Follow the instructions [here](https://open-policy-agent.github.io/gatekeeper/website/docs/install).

    cd <policy_directory>
    kubectl apply -f <policy_name>.yaml
    kubectl apply -f constraints.yaml

**Verifying installed Constraint Templates and Constraints**

    kubectl get constrainttemplates
    kubectl get constraints

**Deleting Constraints and Constraint Templates**

    kubectl delete contraint <constraint_name>
    kubectl delete constrainttemplate <constraint_template_name>

# Library Folders

This section explains the purpose of the policies contained in each folder. It is listed according to the folder names.

## debugging

This folder contains policies that blocks all MongoDB and MongoDBOpsManager resources. It can be used to log all the review objects on the admission controller and you can use the output to craft your own policies. This is explained [here](https://open-policy-agent.github.io/gatekeeper/website/docs/debug).

## mongodb_allow_replicaset

This folder contains policies that only allows MongoDB replicasets to be deployed

## mongodb_allowed_versions

This folder contains policies that only allow specific MongoDB versions to be deployed

## mongodb_strict_tls

This folder contains policies that only allows strict TLS mode for MongoDB deployments

## ops_manager_allowed_versions

This folder contains policies that only allows specific Ops Manager versions to be deployed

## ops_manager_replica_members

This folder contains policies that locks the appDB members and the Ops Manager replicas to a certain number

## ops_manager_wizardless

This folder contains policies that only allows wizardless installation of Ops Manager