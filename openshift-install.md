# OpenShift Install

The MongoDB Enterprise Operator requires two images to work: `operator` and `database` images. The Openshift
installation requires images to be based on Red Hat Enterprise Linux, and these images are published to [Red Hat
Container Catalog](https://catalog.redhat.com/software/containers/explore/). You will have to create special credentials
for your OpenShift installation to be able to fetch images from this registry.

## Create your OpenShift Secret

First, complete the instructions
[here](https://access.redhat.com/terms-based-registry/#/token/openshift3-test-cluster/docker-config). Unfortunatelly,
these instructions refer to a `registry.redhat.io` Registry which is not the one we need, but they accept the same
credentials. First, click on "view its contents" to display the contents we need, and save these contents into a json
file. This file includes 1 entry for `registry.redhat.io`; replicate that entry with a new name,
"`registry.connect.redhat.com`", as in the following example:

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

Now save this file as `dockerconfig` and encode it as a base64 string.

```
$ cat dockerconfig | base64 -w0 > .dockerconfigjson
```

Finally, create a `Secret` object that contains this encoded string:

```
$ kubectl -n <your-namespace> create secret generic openshift-pull-secrets --from-file=.dockerconfigjson
```

## Use the new Secret to pull images

Now that the `Secret` has been created, you need to reference it from the `mongodb-enterprise-openshift.yaml` file.
When you edit this file, you'll realize that there's a `Deployment` object at the end (the one with name
`enterprise-operator`). This `Deployment` needs to be modified slightly, under the `spec` section you need to add
a new attribute, with name `imagePullSecrets` and use the name of the `Secrets` object that you downloaded and created.
The `spec` section will look something like:

```yaml
# ...

spec:
  imagePullSecrets: openshift-pull-secrets  # this is where the name of the Secret goes
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
$ kubectl -n <your-namespace> -f mongodb-enterprise-openshift.yaml
```

From now on, the OpenShift cluster will be authenticated to pull images from the Red Hat registry. Now you should be
able to return to the regular instructions for Kubernetes.
