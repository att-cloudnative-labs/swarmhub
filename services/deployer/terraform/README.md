### Requirements

- [terraform](https://terraform.io)
- [rke](https://rancher.com/docs/rke/latest/en/installation/)


## How to use
```console
#clone this repo
$ git clone https://mingtts@bitbucket.org/kloudius/deployer.git
$ cd deployer

#set API keys to environment variables
$ export AWS_ACCESS_KEY_ID="<your-access-key>"
$ export AWS_SECRET_ACCESS_KEY="<your-secret-key>" 
```

#### provision
```
GRID_NAME=testgrid \
GRID_REGION=us-west-2 \
MASTER_INSTANCE=t2.micro \
SLAVE_INSTANCE=t2.micro \
SLAVE_INSTANCE_CORE=1 \
SLAVE_INSTANCE_COUNT=1 \
PROVISION=true \
PROVIDER=aws \
./setup.sh
```

#### deployment (start test/change locust user count)
```
GRID_NAME=testgrid \
GRID_REGION=us-west-2 \
LOCUST_COUNT=100 \
HATCH_RATE=100 \
SCRIPT_KEY=test-1.zip \
DEPLOYMENT=true \
./setup.sh
```

#### destroy all
```
GRID_NAME=testgrid \
GRID_REGION=us-west-2 \
DESTROY=true \
PROVIDER=aws \
./setup.sh
```

#### destroy deployment (stop test)
```
GRID_NAME=testgrid \
GRID_REGION=us-west-2 \
DESTROY_DEPLOYMENT=true \
./setup.sh
```