# Rancher & Tilt

To use Tilt to develop Rancher do the following:

1. Create a file in the root of the repo called **kind-rancher-dev.yaml**
2. Add the following as contents:

```yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: rancher-dev
nodes:
- role: control-plane
  image: kindest/node:v1.24.7@sha256:577c630ce8e509131eab1aea12c022190978dd2f745aac5eb1fe65c0807eb315
```

3. Create a file in the root of the repo called **tilt-settings.json**
4. Add the following as contents (change where needed to meet your needs):

```json
{
    "default_registry": "ghcr.io/richardcase",
    "debug": {
        "manager": {
            "continue": false,
            "port": 30001
        }
    }
}
```

5. Make the following changes to the chart:
   1. in `chart/Chart.yaml` change **%APP_VERSION%** and **%VERSION%** to be `0.0.0`
   2. in `chart/values.yaml` delete **%POST_DELETE_IMAGE_NAME%** and **%POST_DELETE_IMAGE_TAG%**

> IMPORTANT: don't commit these changes to the repo as they are only required to run locally.

6. Create a kind cluster and start tilt by running the following in a terminal window

```shell
kind create cluster --config kind-rancher-dev.yaml && tilt up
```

> It will be changed to use k3d in the future

7. Press **Space** and see the Tilt up start
