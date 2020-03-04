#!/bin/bash
set -e

if [[ -z $AWS_ACCESS_KEY_ID ]]; then
    if [[ -z $AWS_ACCESS_KEY ]]; then
        echo '$AWS_ACCESS_KEY & $AWS_ACCESS_KEY_ID are undefined'
        exit 1
    else
        export AWS_ACCESS_KEY_ID=$AWS_ACCESS_KEY
    fi
fi

if [[ -z $AWS_DEFAULT_REGION ]]; then
    if [[ -z $AWS_S3_REGION ]]; then
        echo '$AWS_DEFAULT_REGION & $AWS_S3_REGION are undefined'
        exit 1
    else
        export AWS_DEFAULT_REGION=$AWS_S3_REGION
    fi
fi

export KUBERNETES_SERVICE_HOST=

exec "$@"
