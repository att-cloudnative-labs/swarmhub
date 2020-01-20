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


#### Kubernetes Configs and Secrets
In order to deploy deployer we need to ensure the necessary configs and secrets are made.
The jwt-key is what will be used to authenticate json web tokens that will be used for authentication.
```
openssl rand -base64 33 > ./jwt-key
kubectl create secret generic jwt-key --from-file=./jwt-key --namespace=swarmhub
```
Generate localusers.csv file from [pwgen](pwgen/README.md) and create secret
```
kubectl create secret generic localusers --from-file=./localusers.csv --namespace=swarmhub
```
Generate cloud-credentials secret
```
kubectl create secret generic cloud-credentials --from-literal=aws_access_key=$AWS_ACCESS_KEY --from-literal=aws_secret_access_key=$AWS_SECRET_ACCESS_KEY --from-literal=aws_s3_access_key=$AWS_S3_ACCESS_KEY --from-literal=aws_s3_secret_access_key=$AWS_S3_SECRET_ACCESS_KEY --from-literal=aws_s3_bucket_locustfiles=$AWS_S3_BUCKET_LOCUSTFILES --from-literal=aws_s3_bucket_tfstate=$AWS_S3_BUCKET_TFSTATE --from-literal=aws_s3_region=$AWS_S3_REGION --namespace=swarmhub
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
