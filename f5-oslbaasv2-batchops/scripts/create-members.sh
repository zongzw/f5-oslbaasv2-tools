#!/bin/bash

# This script is used to create multiple members in a pool.
# Before running it, please change the variables between the 2 "========" lines
# Some configuration for member creation in this script is hardcoded, such as ip address. 
# Change them if necessary.

workdir=`cd $(dirname $0); pwd`
source $workdir/batchops.conf

source $openrc

# ====================================================

# subnet, used by 'neutron lbaas-loadbalancer-create <subnet>'
subnet=c4cca9f5-83b3-4f1a-a170-baf6cd1104f5

# project_name: the project to create members.
project_name=proj_1

# loadbalancer: the member belongs to.
loadbalancer=lb-1-1

# pool: the member belongs to.
pool=pl-1-1-7

# the range of members, will be used in member's address 
mbrange=26-74

# The ip address first bits.
ip_prefix=1.1.7
# ====================================================

dts=`date +%Y-%m-%d-%H-%M-%S`
source $openrc

# create member
$batchbin --output-filepath $output_dir/create_mb_$dts.json --check-lb $loadbalancer \
    -- --os-project-name $project_name lbaas-member-create --name mb-$ip_prefix-%{mbrange} --subnet $subnet \
        --address $ip_prefix.%{mbrange} --protocol-port 80 $pool \
    ++ mbrange:$mbrange
