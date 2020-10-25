#!/bin/bash

# This script is used to create multiple members in a pool.
# Before running it, please change the variables between the 2 "========" lines
# Some configuration for member creation in this script is hardcoded, such as ip address. 
# Change them if necessary.

cdir=`cd $(dirname $0); pwd`

# ====================================================

# Where to save the result(in json format).
output_dir=$cdir/../output

# openrc file, 'neutron' command depends on, contains 'OS_*' environment variables.
openrc=$cdir/openrc

# The command binary use to execute batch operation, got from: https://github.com/zongzw/f5-oslbaasv2-tools/releases
batchbin=$cdir/../dist/f5-oslbaasv2-batchops-darwin-amd64

# subnet, used by 'neutron lbaas-loadbalancer-create <subnet>'
subnet=private-subnet

# loadbalancer: the member belongs to.
loadbalancer=lb-1

# pool: the member belongs to.
pool=pl-1-1

# the range of members, will be used in member's address 
mbrange=11-40

# ====================================================

dts=`date +%Y-%m-%d-%H-%M-%S`
source $openrc

# create member
$batchbin --output-filepath $output_dir/create_mb_$dts.json --check-lb $loadbalancer \
    -- member-create --name mb-%{mbrange} --subnet %{subnet} \
        --address 11.11.11.%{mbrange} --protocol-port 80 $pool \
    ++ mbrange:$mbrange subnet:$subnet
