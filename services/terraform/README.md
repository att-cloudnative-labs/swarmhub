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
$ export AWS_S3_BUCKET_TFSTATE="<your-bucket-name-for-tfstate>"
$ export AWS_S3_BUCKET_LOCUSTFILES="<your-bucket-name-for-locust-scripts>"
$ export SCRIPT_DIR_PATH="<your-local-terraform-script-path>"
$ export WORKSPACE_DIR="<your-temp-folder-for-terraform-to-use>"
```

#### provision
```
GRID_ID=testgrid \
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
GRID_ID=1a1a97f5-fb0d-4997-bf4d-44d9c5ba7ff5 \
GRID_REGION=us-west-2 \
LOCUST_COUNT=100 \
HATCH_RATE=100 \
SCRIPT_FILE_NAME=locustfile.zip \
SCRIPT_ID=6584406a-152e-49ea-a1a3-360423f9b263 \
TEST_ID=6584406a-152e-49ea-a1a3-360423f9b263 \
DEPLOYMENT=true \
./setup.sh
```

#### destroy all
```
GRID_ID=3dedea92-6706-43c4-a01a-b5f5edf879c1 \
GRID_REGION=us-west-1 \
DESTROY=true \
PROVIDER=aws \
./setup.sh
```

#### destroy deployment (stop test)
```
GRID_ID=testgrid \
GRID_REGION=us-west-2 \
DESTROY_DEPLOYMENT=true \
./setup.sh
```
