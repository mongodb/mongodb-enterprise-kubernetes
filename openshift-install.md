# OpenShift Install

The MongoDB Enterprise Operator requires two images to work: `operator` and `database` images. The Openshift
installation requires images to be based on Red Hat Enterprise Linux, and these images are published to [Red Hat
Container Catalog](https://catalog.redhat.com/software/containers/explore/). You will have to create special credentials
for your OpenShift installation to be able to fetch images from this registry.

## Create your OpenShift Secret

First, complete the instructions
[here](https://access.redhat.com/terms-based-registry/#/token/openshift3-test-cluster/docker-config). Unfortunatelly,
these instructions refer to a `registry.redhat.io` registry which is not the one we need, but they accept the same
credentials. First, click on "Download xxxx-registry-serviceaccount.yaml" and save the yaml file to your disk. Open
the file with your favorite text editor and extract the value corresponding to the key `.dockerconfigjson` into
another file (e.g. `dockerconfig.b64`). Decode the contents of this file.

```
$ base64 -d < dockerconfig.b64 | jq . > dockerconfig.json
```

Edit the resulting file (here `dockerconfig.json`). It should look similar to

```json
{
  "auths": {
    "registry.redhat.io": {
      "auth": "YOURBASE64USERNAMEANDPASSWORD"
    }
  }
}
```

Duplicate the entry with `registry.redhat.io` and replace the registry name by `registry.connect.redhat.com`.
(Dont't forget the comma between the two entries.) You should end up with something like

```json
{
  "auths": {
    "registry.redhat.io": {
      "auth": "YOURBASE64USERNAMEANDPASSWORD"
    },
    "registry.connect.redhat.com": {
      "auth": "YOURBASE64USERNAMEANDPASSWORD"
    }
  }
}
```

Now open the original file `xxxx-registry-serviceaccount.yaml` again and change the name of the secret to
`openshift-pull-secrets`. Replace the `data:` section with `stringData:` and the line starting with
`.dockerconfigjson:` with `.dockerconfigjson: |` followed by the contents of the file
`dockerconfig.json`. You should end up with something like

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: openshift-pull-secrets
stringData:
  .dockerconfigjson: |
    {
      "auths": {
        "registry.redhat.io": {
          "auth": "YOURBASE64USERNAMEANDPASSWORD"
        },
        "registry.connect.redhat.com": {
          "auth": "YOURBASE64USERNAMEANDPASSWORD"
        }
      }
    }
type: kubernetes.io/dockerconfigjson
```

Finally, create a `Secret` object with this yaml file:

```
$ oc create -f xxxx-registry-serviceaccount.yaml
```

The important part is the `type` line. You cannot create this type of secret with `oc create secret ...`.

## Use the new Secret to pull images

Now that the `Secret` has been created, you need to reference it from the `mongodb-enterprise-openshift.yaml` file.
When you edit this file, you'll realize that there's a `Deployment` object at the end (the one with name
`enterprise-operator`). This `Deployment` needs to be modified slightly, under the `spec` section you need to add
a new attribute, with name `imagePullSecrets` and use the name of the `Secrets` object that you downloaded and created.
The `spec` section will look something like:

```yaml
# ...

spec:
  imagePullSecrets:
  - name: openshift-pull-secrets  # this is where the name of the Secret goes
  ...
  containers:
  - name: enterprise-operator
    ...
# ...
```

That's one image. You will also have to set a new environment variable, on the `env` section, like in the following
snippet:

```yaml
containers:
- name: enterprise-operator
  image: registry.connect.redhat.com/mongodb/enterprise-operator:<version>
  imagePullPolicy: Always

  env:
  ...
  - name: IMAGE_PULL_SECRETS
    value: openshift-pull-secrets
  ...
```

## Finish the Operator Installation

Now that we have instructed our OpenShift cluster to be able to fetch images from the Red Hat registry we will be able
to install the operator using:

```bash
$ kubectl -n <your-namespace> apply -f mongodb-enterprise-openshift.yaml
```

From now on, the OpenShift cluster will be authenticated to pull images from the Red Hat registry. Now you should be
able to return to the regular instructions for Kubernetes.
