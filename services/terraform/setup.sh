#!/bin/bash

#AWS_S3_BUCKET_TFSTATE=${AWS_S3_BUCKET_TFSTATE:-swarmhub-tfstate}
#AWS_S3_BUCKET_LOCUSTFILES=${AWS_S3_BUCKET_LOCUSTFILES:-swarmhub-locustfiles}
#AWS_S3_REGION=${AWS_S3_REGION:-us-west-2}
#LOCUST_COUNT=${LOCUST_COUNT:-100}
#HATCH_RATE=${HATCH_RATE:-100}
STOP_TEST=${STOP_TEST:-false}
SCRIPT_DIR_PATH=${SCRIPT_DIR_PATH:-/terraform}
WORKSPACE_DIR=${WORKSPACE_DIR:-/tfworkspace}
ERROR=false

function exitf() {
    printf '%s\n' "$1" >&2     ## Send message to stderr. Exclude >&2 if you don't want it that way.
    rm -rf $WORKSPACE_DIR/$DIR # cleanup terraform templates in home directory
    exit "${2-1}"              ## Return a code specified by $2 or 1 by default.
}

function destroy_grid() {
    terraform destroy -auto-approve \
        -var="grid_id=$GRID_ID" \
        -var="grid_region=$GRID_REGION"
    if [ $? -ne 0 ]; then
        exitf 'failed to destroy grid'
    fi
    aws s3 rm s3://$AWS_S3_BUCKET_TFSTATE/$KEY_BASE-PROVISION
    aws s3 rm s3://$AWS_S3_BUCKET_TFSTATE/$KEY_BASE-BOOTSTRAP
    aws s3 rm s3://$AWS_S3_BUCKET_TFSTATE/$KEY_BASE-DEPLOYMENT
}

function destroy_deployment() {
    terraform destroy -auto-approve \
        -var="bucket_region=$AWS_S3_REGION" \
        -var="bucket_tfstate=$AWS_S3_BUCKET_TFSTATE" \
        -var="tfstate_bootstrap=$KEY_BASE-BOOTSTRAP"
    if [ $? -ne 0 ]; then
        exitf 'failed to destroy deployment'
    fi
    aws s3 rm s3://$AWS_S3_BUCKET_TFSTATE/$KEY_BASE-DEPLOYMENT
}

function deployment_args() {
    $1 -var="locust_count=$LOCUST_COUNT" \
        -var="hatch_rate=$HATCH_RATE" \
        -var="bucket_tfstate=$AWS_S3_BUCKET_TFSTATE" \
        -var="bucket_region=$AWS_S3_REGION" \
        -var="tfstate_bootstrap=$KEY_BASE-BOOTSTRAP" \
        -var="stop_test=$STOP_TEST" \
        -var="test_id"=$TEST_ID
}

function provision_args() {
    $1 \
        -var="grid_id=$GRID_ID" \
        -var="grid_region=$GRID_REGION" \
        -var="ttl=$TTL" \
        -var="master_instance_type=$MASTER_INSTANCE" \
        -var="slave_instance_type=$SLAVE_INSTANCE" \
        -var="slave_instance_count=$SLAVE_INSTANCE_COUNT" \
        -var="slave_instance_core=$(($SLAVE_INSTANCE_CORE * $SLAVE_INSTANCE_COUNT))"
}

function undo() {
    $1 "terraform plan -destroy -out=tfplan -input=false"
    terraform apply -input=false tfplan
    if [ $? -ne 0 ]; then
        exitf 'failed to undo action'
    fi
}

# check if variables are set
if [[ -z $GRID_ID || -z $GRID_REGION || -z $AWS_S3_BUCKET_TFSTATE || -z $WORKSPACE_DIR || -z $SCRIPT_DIR_PATH ]]; then
    exitf 'one or more variables are undefined'
fi

# prepare directory and s3 state key object
DIR=$GRID_ID-$(date "+%Y%m%d-%H:%M:%S")
KEY_BASE=$GRID_ID-$GRID_REGION

# copy the terraform templates to new folder in home directory
mkdir -p $WORKSPACE_DIR
cp -r $SCRIPT_DIR_PATH $WORKSPACE_DIR/$DIR
cd $WORKSPACE_DIR/$DIR
pwd

# switch to correct directory
if [[ "$PROVISION" = "true" || "$DESTROY" = "true" ]]; then
    if [[ -z $PROVIDER ]]; then
        exitf 'one or more variables are undefined'
    fi
    cd provision/$PROVIDER
    KEY=$KEY_BASE-PROVISION

elif [[ "$DEPLOYMENT" = "true" || "$DESTROY_DEPLOYMENT" = "true" ]]; then
    cd deployment
    KEY=$KEY_BASE-DEPLOYMENT
fi

# initialize terraform backend
terraform init -input=false \
    -backend-config="bucket=$AWS_S3_BUCKET_TFSTATE" \
    -backend-config="key=$KEY" \
    -backend-config="region=$AWS_S3_REGION"
if [ $? -ne 0 ]; then
    exitf 'failed to init terraform backend'
fi

# destroy grid
if [ "$DESTROY" = "true" ]; then
    destroy_grid
elif [[ "$DESTROY_DEPLOYMENT" = "true" ]]; then
    destroy_deployment
elif [ "$PROVISION" = "true" ]; then
    # setup grid

    # check if variables are set
    if [[ -z $MASTER_INSTANCE || -z $SLAVE_INSTANCE || -z $SLAVE_INSTANCE_CORE || -z $SLAVE_INSTANCE_COUNT || -z $AWS_S3_BUCKET_LOCUSTFILES || -z $TTL ]]; then
        exitf 'one or more variables are undefined'
    fi

    if [ "$PROVIDER" = "aws" ]; then
        provision_args "terraform apply -auto-approve"

    elif [ "$PROVIDER" = "prem" ]; then
        exitf 'not implemented'
    fi
    if [ $? -ne 0 ]; then
        undo "provision_args"
        aws s3 rm s3://$AWS_S3_BUCKET_TFSTATE/$KEY_BASE-PROVISION
        exitf 'failed to provision grid'
    fi

    # bootstrap kubernetes cluster based on provisioned nodes
    cd $WORKSPACE_DIR/$DIR/bootstrap
    terraform init \
        -backend-config="bucket=$AWS_S3_BUCKET_TFSTATE" \
        -backend-config="key=$KEY_BASE-BOOTSTRAP" \
        -backend-config="region=$AWS_S3_REGION" \
        -input=false
    if [ $? -ne 0 ]; then
        cd $WORKSPACE_DIR/$DIR/provision/$PROVIDER
        undo "provision_args"
        aws s3 rm s3://$AWS_S3_BUCKET_TFSTATE/$KEY_BASE-PROVISION
        exitf 'failed to init terraform backend for bootstrap'
    fi

    terraform apply -auto-approve \
        -var="tfstate_provision=$KEY_BASE-PROVISION" \
        -var="bucket_tfstate=$AWS_S3_BUCKET_TFSTATE" \
        -var="bucket_region=$AWS_S3_REGION"
    if [ $? -ne 0 ]; then
        cd $WORKSPACE_DIR/$DIR/provision/$PROVIDER
        undo "provision_args"
        aws s3 rm s3://$AWS_S3_BUCKET_TFSTATE/$KEY_BASE-PROVISION
        aws s3 rm s3://$AWS_S3_BUCKET_TFSTATE/$KEY_BASE-BOOTSTRAP
        exitf 'failed to bootstrap grid'
    fi

elif [ "$DEPLOYMENT" = "true" ]; then
    # deploy locust

    # check if variables are set
    if [[ -z $LOCUST_COUNT || -z $HATCH_RATE || -z $GRID_ID || -z $GRID_REGION || -z $AWS_S3_BUCKET_LOCUSTFILES || -z $SCRIPT_ID || -z $SCRIPT_FILE_NAME || -z $TEST_ID ]]; then
        exitf 'one or more variables are undefined'
    fi

    aws s3api get-object --bucket $AWS_S3_BUCKET_LOCUSTFILES --key scripts/$SCRIPT_ID/file/$SCRIPT_FILE_NAME $SCRIPT_FILE_NAME
    if [ $? -ne 0 ]; then
        exitf 'failed to download locust script'
    fi
    unzip -o $SCRIPT_FILE_NAME
    if [ $? -ne 0 ]; then
        exitf 'failed to unzip locust script'
    fi
    deployment_args "terraform apply -auto-approve"
    if [ $? -ne 0 ]; then
        undo "deployment_args"
        aws s3 rm s3://$AWS_S3_BUCKET_TFSTATE/$KEY_BASE-DEPLOYMENT
        exitf 'failed to deploy test'
    fi
fi

exitf "" 0
