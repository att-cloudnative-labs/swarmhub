#!/bin/bash
set -e

# This is for amazon deployment
# ${1} is grid id
# ${2} is the grid region
# ${3} is locust master size
# ${4} is locust slave size
# ${5} is number of locust slaves
# ${6} is the grid TTL
# ${7} security groups for master node
# ${8} security groups for slave nodes
# ${9} instance types (spot, etc)

echo "Running locust provisioning script"

export ANSIBLE_HOST_KEY_CHECKING=False
cd /ansible/
echo "Delete Existing Master Node if exists (it shouldn't):"
ansible-playbook gridProvision.yml --extra-vars "region=${2} tag_name=master tag_grid=${1} instance_count=0"

echo "Delete Existing Slave Node if exists (it shouldn't):"
ansible-playbook gridProvision.yml --extra-vars "region=${2} tag_name=slave tag_grid=${1} instance_count=0"

echo "Provision New Master Node:"
ansible-playbook gridProvision.yml --extra-vars="{\"region\": \"${2}\", \"tag_name\": \"master\", \"tag_grid\": \"${1}\", \"tag_ttl\": \"${6}\", \"instance_type\": \"${3}\", \"instance_count\": 1, \"security_groups\": ${7}}"

echo "Provision New Slave Node:"
ansible-playbook gridProvision.yml --extra-vars="{\"region\": \"${2}\", \"tag_name\": \"slave\", \"tag_grid\": \"${1}\", \"tag_ttl\": \"${6}\", \"instance_type\": \"${4}\", \"instance_count\": ${5}, \"security_groups\": ${8}}"
