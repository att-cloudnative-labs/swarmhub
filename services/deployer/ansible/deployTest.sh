#!/bin/bash
set -e

# This is for amazon deployment
# ${1} is script id
# ${2} is the test filename
# ${3} is grid id
# ${4} is the grid region
# ${5} is does the grid start automatically?

echo "Running locust deployTest script"

export ANSIBLE_HOST_KEY_CHECKING=False
echo "cd into ansible folder."
cd /ansible/

echo "Load locust test files"
ansible-playbook deployTest.yml --extra-vars "region=${4} grid_id=${3} start_automatically=${5} script_id=${1} script_filename=${2}" --private-key /root/.ssh/swarmhub.pem 2>&1
