Step 1: Create a namespace to install Mongo DB

oc create ns mongodb


Step 2: Install the operator in the cluster in the namespace created above


Step 3: Wait for the Operator to be deployed.

Step 4: The links that we find on the OCP page are not the links to the instructions we need to create the MongoDB Ops Manager. 

Documentation link: https://docs.mongodb.com/kubernetes-operator/stable/tutorial/install-k8s-operator/

Create a Secret... link: https://docs.mongodb.com/kubernetes-operator/stable/tutorial/install-k8s-operator/#create-credentials

Create a ConfigMap... link: https://docs.mongodb.com/kubernetes-operator/stable/tutorial/install-k8s-operator/#create-onprem-project


The link that was helpful to find the command to create the prequisite setp secret: https://docs.mongodb.com/kubernetes-operator/stable/tutorial/plan-om-resource/#prerequisites
Use the link above to find the information on how to proceed with the operator deployment


Step 5: Pre-requisite step to create the secret. (There was no mention of this step to create the secret in OCP.)

https://docs.mongodb.com/kubernetes-operator/stable/tutorial/plan-om-resource/#prerequisites
Follow the instructions on Step4 to create the secret. These are kubectl commands, but we can use oc commands. 

Make sure you give a password that meets the mongodb requirements. Please note the name of the secret created here. I have created it with the name ops-manager-admin
  oc create secret generic ops-manager-admin \
  --from-literal=Username="admin" \
  --from-literal=Password="Passw0rd@2020" \
  --from-literal=FirstName="sayari" \
  --from-literal=LastName="mukherjee"

Step 6: Creating MongoDB Ops Manager Instance
The next step is to create the MongoDB Ops Manager Instance. Click on the 3rd tile to Create Instance

Change the adminCredentials value to point to the name of the secret created in Step 5. In my case it is ops-manager-admin. Click create. 

Step 7: Verify MongoDB Ops Manager is successfully deployed. Verify resources of the Ops Manager, OpsManager Pods to be Running, Secrets to get created. (This step was also not clear in documentation.)

***NOTE: Wait for the secret ops-manager-admin-key to be created. It has the value for user and publicApiKey required in the subsequent steps***

Step 8: Creating Secret and ConfigMap

Instead of the links in the OCP console, follow the links here

Creating Secret:
https://docs.mongodb.com/kubernetes-operator/stable/tutorial/create-operator-credentials/#create-k8s-credentials

We can use oc commands instead. Give a name for this secret. Find the value for user and publicApiKey from the secret that gets created - ***ops-manager-admin-key***

oc -n mongodb \
  create secret generic mongodb-creds \
  --from-literal="user=admin" \
  --from-literal="publicApiKey=bfaf1b27-e261-4e6e-a0e5-ad1783a7e8c7"


Creating ConfigMap: https://docs.mongodb.com/kubernetes-operator/stable/tutorial/create-project-using-configmap/

We can use oc commands. Give a name to the ConfigMap.

Find the service name and port for OpsManager. ops-manager-svc. 

Build the baseUrl: http://<ops-manager-service-name>:<port>. In this case : http://ops-manager-svc:8080

We don't need the optional parameter. Run the command in cluster. 


  oc create configmap mongodb-cm \
  --from-literal="baseUrl=http://ops-manager-svc:8080"


Step 9: Create the MongoDB Deployment Instance

Click on the first tile to create the MongoDB Deployment Instance

Substitute the values for spec.credentials giving the Secret created above - mongodb-creds and the ConfigMap created above - mongodb-cm. Click Create. 

Step 10: Verify Operand Installed successfully:

Verify Pods: verify replicate set pods to be running
Verify Stateful Sets
Verify Services

Step 11: Verify connection to OpsManager
Create a route to ops-manager-svc-ext

Click on location url to launch the Ops Manager

Step 12: Verify Application
We can create a route for my-replica-set-svc

From a mongo client try connecting user username and password. Watch for the Pod logs.

