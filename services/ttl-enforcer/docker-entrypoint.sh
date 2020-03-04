#!/bin/sh
set -e

if [[ -z $AWS_DEFAULT_REGION ]]; then
    if [[ -z $AWS_S3_REGION ]]; then
        echo '$AWS_DEFAULT_REGION & $AWS_S3_REGION are undefined'
        exit 1
    else
        export AWS_DEFAULT_REGION=$AWS_S3_REGION
    fi
fi

exec "$@"
