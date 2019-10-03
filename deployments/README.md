## Getting up and running
#### High Level Steps for Deploying
1. Have a running [kubernetes cluster](https://docs.aws.amazon.com/eks/latest/userguide/getting-started-eksctl.html) and create a swarmhub namespace
2. Deploy a [nats streaming cluster](https://github.com/nats-io/nats-streaming-operator)
3. Deploy a [cockroachDB cluster](https://www.cockroachlabs.com/docs/stable/orchestrate-cockroachdb-with-kubernetes.html)
4. Initialize the cockroachDB cluster with data from [here](db/tables.txt)
5. Deploy deployer
6. Deploy ttl-enforcer
7. Deploy swarmhub
8. Deploy ingress

#### AWS Region Changes
Make sure the region has default subnets, if they don't create them
```
aws ec2 create-default-subnet --region us-east-2 --availability-zone us-east-2a
aws ec2 create-default-subnet --region us-east-2 --availability-zone us-east-2b
aws ec2 create-default-subnet --region us-east-2 --availability-zone us-east-2c
...
```

The needed security groups if you aren't passing in your own custom are `default`, `swarmhub_prometheus`, `swarmhub_SSH`, and `swarmhub_HTTPS`. `default` gives VMs permissions to access eachother, `swarmhub_prometheus` opens up the VMs to be hit by your prometheus instance, `swarmhub_SSH` is used to give SSH access to all the VMs, `swarmhub_HTTPS` is added to the locust masters so you can go to the locust UI. You will need to create these for each of the regions you plan on deploying load tests into.

An example of creating security groups (note you will want to better limit cidr access based on your environment):
```
## To get vpc-id information
aws ec2 describe-vpcs

## HTTPS
aws ec2 create-security-group --group-name swarmhub_HTTPS --description "Give HTTPS access" --vpc-id vpc-yourID
aws ec2 authorize-security-group-ingress --group-name swarmhub_HTTPS --protocol tcp --port 443 --cidr 0.0.0.0/0

## PROMETHEUS
aws ec2 create-security-group --group-name swarmhub_prometheus --description "To allow prometheus to scrape the VMs" --vpc-id vpc-yourID
aws ec2 authorize-security-group-ingress --group-name swarmhub_prometheus --protocol tcp --port 8080 --cidr 0.0.0.0/0
aws ec2 authorize-security-group-ingress --group-name swarmhub_prometheus --protocol tcp --port 19999 --cidr 0.0.0.0/0
aws ec2 authorize-security-group-ingress --group-name swarmhub_prometheus --protocol tcp --port 443 --cidr 0.0.0.0/0

## SSH
aws ec2 create-security-group --group-name swarmhub_SSH --description "To enable SSH access" --vpc-id vpc-yourID
aws ec2 authorize-security-group-ingress --group-name swarmhub_SSH --protocol tcp --port 22 --cidr 0.0.0.0/0
```

Import the ssh key `swarmhub` into the region.
```
aws ec2 import-key-pair --key-name swarmhub --public-key-material "$(ssh-keygen -y -f ~/.ssh/swarmhub.pem)"
```

#### Get a working Kubernetes Cluster
Follow instructions [here](https://docs.aws.amazon.com/eks/latest/userguide/getting-started-eksctl.html) for eks.

#### Generate the Namespace
First step to deploying swarmhub would be to create the namespace.
```
{
  "apiVersion": "v1",
  "kind": "Namespace",
  "metadata": {
    "name": "swarmhub",
    "labels": {
      "name": "swarmhub"
    }
  }
}
```
```
kubectl create -f namespace.json
```

#### Generating the AWS AMI
A custom AMI is needed for the grid deployments. We use a custom AMI instead of one of the defaults is to speed up deployment times by baking in the programs we need. Start off with a centos base image which you can find one [here](https://wiki.centos.org/Cloud/AWS).
Deploy an instance and ssh into it so we can make the necessary changes.
  
Install netdata for monitoring, instructions [here](https://github.com/netdata/netdata#quick-start).
Install a virtual environment and install locust and prometheus:
```
sudo yum install -y https://centos7.iuscommunity.org/ius-release.rpm
sudo yum update -y
sudo yum install -y python36u python36u-libs python36u-devel python36u-pip zip unzip
sudo mkdir /opt/python
sudo chown centos:centos /opt/python
mkdir /opt/python/locust
python3.6 -m venv /opt/python/venv
source /opt/python/venv/bin/activate
pip install locustio
pip install prometheus_client
deactivate
exit
```
Go to AWS EC2 and Right click on the instance and click on Image ->  Create Image. Save this AMI ID for when configuring the ansible configs for the deployer. Copy over the locust AMI to all the regions you plan on deploying load tests to.


#### Kubernetes Configs and Secrets
In order to deploy deployer we need to ensure the necessary configs and secrets are made.
The jwt-key is what will be used to authenticate json web tokens that will be used for authentication.
```
openssl rand -base64 33 > ./jwt-key
kubectl create secret generic jwt-key --from-file=./jwt-key --namespace=swarmhub
```
A pem file will need to also be uploaded, this is how the deployer will be able to ssh into the vms.
```
kubectl create secret generic ssh --from-file=./swarmhub.pem --namespace=swarmhub
```
Generate localusers.csv file from [pwgen](pwgen/README.md) and create secret
```
kubectl create secret generic localusers --from-file=./localusers.csv --namespace=swarmhub
```
Generate cloud-credentials secret
```
kubectl create secret generic cloud-credentials --from-literal=aws_access_key=$AWS_ACCESS_KEY --from-literal=aws_secret_access_key=$AWS_SECRET_ACCESS_KEY --from-literal=aws_s3_access_key=$AWS_S3_ACCESS_KEY --from-literal=aws_s3_secret_access_key=$AWS_S3_SECRET_ACCESS_KEY --from-literal=aws_s3_bucket=$AWS_S3_BUCKET --from-literal=aws_s3_region=$AWS_S3_REGION --namespace=swarmhub
```
Generate TLS information and create secret.
```
openssl req \
    -new \
    -newkey rsa:4096 \
    -days 36500 \
    -nodes \
    -x509 \
    -subj "/C=US/ST=CA/L=Malibu/O=My Company/CN=Swarmhub" \
    -keyout server.key \
    -out server.crt
kubectl create secret generic tls --from-file=./server.key --from-file=./server.crt --namespace=swarmhub  
```
Load AMI information into deployer-configs
```
kubectl -n swarmhub create configmap deployer-configs --from-literal=aws_us_east_1_ami=$AWS_US_EAST_1_AMI --from-literal=aws_us_east_2_ami=$AWS_US_EAST_2_AMI --from-literal=aws_us_west_1_ami=$AWS_US_WEST_1_AMI --from-literal=aws_us_west_2_ami=$AWS_US_WEST_2_AMI
```

#### Deploying swarmhub services
Build and deploy a docker image for the services and then in the k8s yaml files replace `#build an image and put here` with your images. Then run the following kubectl commands:
```
kubectl --namespace swarmhub apply -f k8s-files/swarmhub.yaml
kubectl --namespace swarmhub apply -f k8s-files/deployer.yaml
kubectl --namespace swarmhub apply -f k8s-files/ttl-enforcer.yaml
```

#### Setup an ingress for swarmhub
An example of an ingress deployment can be seen [here](https://docs.aws.amazon.com/eks/latest/userguide/alb-ingress.html). This example doesn't consider TLS and services that are using self signed certs. Setting up an ingress is beyond the scope of this README.
  
If an ingress is not set up you can port-forward swarmhub with the following command:
```
kubectl -n=swarmhub port-forward $(kubectl -n=swarmhub get po | grep swarmhub- | awk '{print $1;}') 8443:8443
```
And then access by visiting https://localhost:8443
