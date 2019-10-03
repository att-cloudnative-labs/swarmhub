# Deployer
Deployer handles the deployments of the load tests. It is a wrapper around Ansible. There are ansible roles for provisioning, deploying, as well as deleting a grid.

## Configuring 
For the deployer to work properly you need to set the necessary ansible variables. You need to set the "ami" variable in the grid-cleanup role. They are commented out by default.

```
.
├── ansible
│   └── roles
│       └── grid-cleanup
│           └── vars
│               ├── us-east-1.yml
│               ├── us-east-2.yml
│               ├── us-west-1.yml
│               └── us-west-2.yml
```