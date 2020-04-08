## How to build

### Preequistes
Put the server certificates `server.key` and `server.crt` into `terraform/bootstrap` directory. Make sure the certificate is same as swarmhub web certificates. Please refer to [deployments](../deployments/README.md) for `server.key` and `server.crt` creation.

#### Deployer/TTL-Enforcer (build in current folder as ./terraform is shared between both services)
```
docker build --no-cache -t <deployer/ttl-enforcer> -f  <deployer/ttl-enforcer>/Dockerfile .
```

#### Swarmhub (build in ./swarmhub folder)
```
cd swarmhub
docker build --no-cache -t swarmhub .
```