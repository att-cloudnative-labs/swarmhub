# Helmfile Deployment

**Note: This is only for development environment**

## Prerequistes
* [Helmfile](https://github.com/roboll/helmfile) with helm-diff plugin installed
* [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)

## Folder description
```
cockroachdb                   CockroachDB service (database)
grafana                       Grafana service (for monitoring dashboard)
nats-streaming-operator       NATS streaming service (messaging protocol)
prometheus                    Prometheus service (metrics scraping)
swarmhub                      Swarmhub services (swarmhub, ttl-enforcer, deployer)
```

# Deployment
## Pre-deployment steps
**Make sure your `kubectl` command is pointing to your desired cluster before running `helmfile`.**

**Create namespace `swarmhub` before applying any helmfile.**

## CockroachDB
### Default Spec
* 1 replica
* local Persistent Volume Path: `/data/cockroachdb_pv_0`

### Notes
* Make sure in the k8s cluster host `/data/cockroachdb_pv_0` directory exist before installation.
* After the service is up, populate the initial database based on [tables.txt](../db/tables.txt).
* `helmfile destroy` might not clean up properly, need to manually delete the PersistentVolumeClaim using `kubectl -n swarmhub delete pvc  	datadir-cockroachdb-0`

## NATS Streaming Operator
### Default Spec
* 1 replica for stan
* 1 replica for nats
* local Persistent Volume Path: `/data/stan`

### Notes
* Make sure in the k8s cluster host `/data/stan` directory exist before installation.
* It will fail on the first time installation as the CRD is not created yet. Try to reinstall nats-streaming operator again by running `helmfile destroy && helmfile apply`.
* `helm destroy` will not cleanup `nats-x` and `stan-x` properly. You need to remove that manually.

## Swarmhub
### Default Spec
* 1 replica for swarmhub
* 3 replica for deployer
* 1 replica for ttl-enforcer
* Using local docker image (If using local docker images, make sure you can SSH into the kubernetes cluster host machine to run docker build)

### Deployment steps
1. Create `jwt-key`, `localusers.csv`, `server.crt` & `server.key` based on [deployments](../README.md) and put them in `helmfile/swarmhub/init/files` directory.
2. Create an `sh` file in the directory `helmfile/swarmhub/init/files` (e.g. `setenv.sh` ) with the following syntax:
```
#!/bin/bash

export AWS_ACCESS_KEY=
export AWS_SECRET_ACCESS_KEY=
export AWS_S3_ACCESS_KEY=
export AWS_S3_SECRET_ACCESS_KEY=
export AWS_S3_BUCKET_LOCUSTFILES=
export AWS_S3_BUCKET_TFSTATE=
export AWS_S3_REGION=
```
3. Run `. ./swarmhub/init/files/setenv.sh` to set the environment variables.
4. Go to `helmfile/swarmhub` and run `helmfile apply` to deploy all the swarmhub services (`swarmhub`, `ttl-enforcer` & `deployer`).
5. The web will be served at `https://<URL>:30000`

### Notes
* If you have changes any secret or configmap, reinstall the swarmhub-init by running `helmfile apply` at `helmfile/swarmhub/init`
* To deploy individual services, go to respective folder and run `helmfile apply`.

## Prometheus
### Default Spec
* local Persistent Volume Path: `/data/prometheus-pv`

### Notes
* Make sure in the k8s cluster host `/data/prometheus-pv` directory exist before installation.

## Grafana
### Default Spec
* local Persistent Volume Path: `/data/grafana-pv`

### Notes
* Make sure in the k8s cluster host `/data/grafana-pv` directory exist before installation.


# Command Examples
```
# Install the service
helmfile apply                        // current directory
helmfile -f prometheus/ apply         // By path

# Uninstall the service
helmfile destroy                      // current directory
helmfile -f prometheus/ destroy       // By path

# One-liner uninstall and reinstall the service (Normally, when the docker images on remote side need to be updated)
helmfile destoy && helmfile apply                                 // current directory
helmfile -f swarmhub/ destroy && helmfile -f swarmhub/ apply      //By path

```

* `helmfile apply` also will detect if there is any changes in helmfile or k8s files and apply the approapriate changes to the cluster.