## How to build

#### Deployer/TTL-Enforcer (build in current folder as ./terraform is shared between both services)
```
docker build --no-cache -t <deployer/ttl-enforcer> -f  <deployer/ttl-enforcer>/Dockerfile .
```

#### Swarmhub (build in ./swarmhub folder)
```
cd swarmhub
docker build --no-cache -t swarmhub .
```