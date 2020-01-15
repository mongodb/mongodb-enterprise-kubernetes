# OpenShift Install

The MongoDB Enterprise Operator requires two images to work: `operator` and `database`. The Openshift installation requires images to be based on Red Hat Enterprise Linux and these images are published on [Red Hat Container Catalog](https://catalog.redhat.com/software/containers/explore/).

## Create your OpenShift Secret

Prepare a secret with your credentials to be able to pull images from the registries `registry.redhat.io` and `registry.connect.redhat.com`. To achieve this visit [https://access.redhat.com/terms-based-registry/](https://access.redhat.com/terms-based-registry/), choose the appropriate account and download the OpenShift pull secret (to be found under the *OpenShift Secret* tab). Let's assume its name is `7654321_mycompany-registry-credentials-secret.yaml`. This secret has one entry with key `.dockerconfigjson`. The value is base64 encoded. Use your favorite text editor to extract this value into a separate file. Let's call it `dockerconfig.b64`. Decode the value with

```
base64 -d < dockerconfig.b64 | jq . > dockerconfig.json
```

The resulting file `dockerconfig.json` should look similar to

```json
{
  "auths": {
    "registry.redhat.io": {
      "auth": "RNVpqSTBPVEV3WldZMFl6ZGh..."
    }
  }
}
```

This is the access token needed to access `registry.redhat.io`. In order to be able to also access `registry.connect.redhat.com` duplicate the entry with name `registry.redhat.io` and change the key to `registry.connect.redhat.com`. You should end up with something like

```json
{
  "auths": {
    "registry.redhat.io": {
      "auth": "RNVpqSTBPVEV3WldZMFl6ZGh..."
    },
    "registry.connect.redhat.com": {
      "auth": "RNVpqSTBPVEV3WldZMFl6ZGh..."
    }
  }
}
```

Don't forget the comma between the two entries under `auths`!

We now create the definition of an OpenShift secret with the above JSON document as value.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: openshift-pull-secret
stringData:
  .dockerconfigjson: |
    {
      "auths": {
        "registry.redhat.io": {
          "auth": "RNVpqSTBPVEV3WldZMFl6ZGh..."
        },
        "registry.connect.redhat.com": {
          "auth": "RNVpqSTBPVEV3WldZMFl6ZGh..."
        }
      }
    }
type: kubernetes.io/dockerconfigjson
```

We assume that you store this YAML file as `openshift-pull-secret.yaml`. Finally create this secret with

```
oc create -f openshift-pull-secret.yaml
```

## Install the MongoDB Enterprise Operator

Create a set of OpenShift resources with

```bash
oc create -f mongodb-enterprise-openshift.yaml
```

Make sure the pod starts properly. The MongoDB Enterprise Operator is now in place.

The service accounts defined in the above `mongodb-enterprise-openshift.yaml` are linked to the pull secret we created earlier. The service account *default* (automatically created for every project) needs to be linked manually.

```
oc secret link default openshift-pull-secret --for=pull
```

Now you should be able to return to the regular [instructions for Kubernetes](mongodb-enterprise-kubernetes#mongodb-object).
