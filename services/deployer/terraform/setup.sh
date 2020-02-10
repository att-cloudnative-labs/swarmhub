#!/bin/sh

#BUCKET_TFSTATE=${BUCKET_TFSTATE:-swarmhub-tfstate}
#BUCKET_LOCUSTFILES=${BUCKET_LOCUSTFILES:-swarmhub-locustfiles}
#BUCKET_REGION=${BUCKET_REGION:-us-west-2}
#LOCUST_COUNT=${LOCUST_COUNT:-100}
#HATCH_RATE=${HATCH_RATE:-100}
STOP_TEST=${STOP_TEST:-false}
#SCRIPT_DIR_PATH=${SCRIPT_DIR_PATH:-./}
#WORKSPACE_DIR=${WORKSPACE_DIR:-/tfworkspace}

# check if variables are set
if [[ -z $GRID_NAME || -z $GRID_REGION || -z $BUCKET_TFSTATE || -z $WORKSPACE_DIR || $SCRIPT_DIR_PATH || -z $WORKSPACE_DIR ]]; then
    echo 'one or more variables are undefined'
    exit 1
fi

# prepare directory and s3 state key object
DIR=$GRID_NAME-$(date "+%Y%m%d-%H:%M:%S")
KEY_BASE=$GRID_NAME-$GRID_REGION

# copy the terraform templates to new folder in home directory
mkdir -p $WORKSPACE_DIR
cp -r $SCRIPT_DIR_PATH $WORKSPACE_DIR/$DIR
cd $WORKSPACE_DIR/$DIR
pwd

# switch to correct directory
if [[ "$PROVISION" = "true" || "$DESTROY" = "true" ]]; then
    if [[ -z $PROVIDER ]]; then
        echo 'one or more variables are undefined'
        exit 1
    fi
    cd provision/$PROVIDER
    KEY=$KEY_BASE-PROVISION

elif [[ "$DEPLOYMENT" = "true" || "$DESTROY_DEPLOYMENT" = "true" ]]; then
    cd deployment
    KEY=$KEY_BASE-DEPLOYMENT
fi

# initialize terraform backend
terraform init \
    -backend-config="bucket=$BUCKET_TFSTATE" \
    -backend-config="key=$KEY" \
    -backend-config="region=$BUCKET_REGION" \
    -input=false

# destroy grid
if [ "$DESTROY" = "true" ]; then
    terraform destroy -auto-approve \
        -var="grid_name=$GRID_NAME" \
        -var="grid_region=$GRID_REGION"
    if [ !$? ]; then
        aws s3 rm s3://$BUCKET_TFSTATE/$KEY_BASE-PROVISION
        aws s3 rm s3://$BUCKET_TFSTATE/$KEY_BASE-BOOTSTRAP
        aws s3 rm s3://$BUCKET_TFSTATE/$KEY_BASE-DEPLOYMENT
    fi
elif [[ "$DESTROY_DEPLOYMENT" = "true" ]]; then
    terraform destroy -auto-approve \
        -var="bucket_region=$BUCKET_REGION" \
        -var="bucket_tfstate=$BUCKET_TFSTATE" \
        -var="tfstate_provision=$KEY_BASE-PROVISION"

    if [ !$? ]; then
        aws s3 rm s3://$BUCKET_TFSTATE/$KEY_BASE-DEPLOYMENT
    fi
elif [ "$PROVISION" = "true" ]; then
    # setup grid

    # check if variables are set
    if [[ -z $LOCUST_COUNT || -z $HATCH_RATE || -z $MASTER_INSTANCE || -z $SLAVE_INSTANCE || -z $SLAVE_INSTANCE_CORE || -z $SLAVE_INSTANCE_COUNT || -z $BUCKET_LOCUSTFILES ]]; then
        echo 'one or more variables are undefined'
        exit 1
    fi

    if [ "$PROVIDER" = "aws" ]; then
        terraform apply -auto-approve \
            -var="grid_name=$GRID_NAME" \
            -var="grid_region=$GRID_REGION" \
            -var="master_instance_type=$MASTER_INSTANCE" \
            -var="slave_instance_type=$SLAVE_INSTANCE" \
            -var="slave_instance_count=$SLAVE_INSTANCE_COUNT" \
            -var="slave_instance_core=$(($SLAVE_INSTANCE_CORE * $SLAVE_INSTANCE_COUNT))"
    elif [ "$PROVIDER" = "prem" ]; then
        terraform apply -auto-approve \
            -var="private_key=$PRIVATE_KEY" \
            -var="ssh_username=$SSH_USERNAME" \
            -var="kube_master_ip=$KUBE_MASTER_IP" \
            -var="locust_master_ip=$LOCUST_MASTER_IP" \
            -var="locust_slave_ips=($LOCUST_SLAVE_IPS)" \
            -var="slave_instance_core=$(($SLAVE_INSTANCE_CORE * $SLAVE_INSTANCE_COUNT))"
    fi

    # bootstrap kubernetes cluster based on provisioned nodes
    cd $WORKSPACE_DIR/$DIR/bootstrap
    terraform init \
        -backend-config="bucket=$BUCKET_TFSTATE" \
        -backend-config="key=$KEY_BASE-BOOTSTRAP" \
        -backend-config="region=$BUCKET_REGION" \
        -input=false

    terraform apply -auto-approve \
        -var="tfstate_provision=$KEY_BASE-PROVISION" \
        -var="bucket_tfstate=$BUCKET_TFSTATE" \
        -var="bucket_region=$BUCKET_REGION"

elif [ "$DEPLOYMENT" = "true" ]; then
    # deploy locust

    # check if variables are set
    if [[ -z $LOCUST_COUNT || -z $HATCH_RATE || -z $GRID_NAME || -z $GRID_REGION || -z $BUCKET_LOCUSTFILES || -z $SCRIPT_KEY ]]; then
        echo 'one or more variables are undefined'
        exit 1
    fi

    aws s3api get-object --bucket $BUCKET_LOCUSTFILES --key $SCRIPT_KEY $SCRIPT_KEY

    unzip -o $SCRIPT_KEY

    terraform apply -auto-approve \
        -var="locust_count=$LOCUST_COUNT" \
        -var="hatch_rate=$HATCH_RATE" \
        -var="bucket_tfstate=$BUCKET_TFSTATE" \
        -var="bucket_region=$BUCKET_REGION" \
        -var="tfstate_bootstrap=$KEY_BASE-BOOTSTRAP" \
        -var="stop_test=$STOP_TEST"

fi

# cleanup terraform templates in home directory
rm -rf $WORKSPACE_DIR/$DIR
