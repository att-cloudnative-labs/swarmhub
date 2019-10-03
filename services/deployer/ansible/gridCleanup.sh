#!/bin/bash
set -e

# This is for amazon deployment deletion
# ${1} is grid id
# ${2} is the grid region

echo "Running grid cleanup script"

export ANSIBLE_HOST_KEY_CHECKING=False
cd /ansible/
echo "Cleanup Existing Grid Nodes ${1}:"
ansible-playbook gridCleanup.yml --extra-vars "region=${2} tag_grid=${1}"
