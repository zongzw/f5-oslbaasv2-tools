# Ansible Playbooks For Performance Test

Files in this folder is another way for creating lbaasv2 resources in batch, compared to `f5-oslbaasv2-batchops`. 

## Why Do we need It

`f5-oslbaasv2-batchops` is an executable binary, which is not easy for maintenance, also the parameter is relatively complicated.

## Usage

### 1. Preparation

```
-> [Run once] $ ansible-playbook -i env-lab.ini -e@vars.yml create-projects-networks.yml
```

This command is used to create projects and networks(including subnets) before resource creation.

The count to be created is defined in vars-XX.yml file, like `projects: 200`

### 2. Resource Creation

```
-> [Run repeatablely] $ ansible-playbook -i env-lab.ini -e@vars.yml create-resources-in-batch.yml -e index=<num>
```

This command is used to create resources under project `<num>`
the `<num>` is the project index.

Read ansible scripts in this folder for more details.