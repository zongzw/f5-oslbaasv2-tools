#!/bin/bash

# This is a performance test script.
# In sequence, it will create loadbalancer > pool > listener > member > healthmonitor > l7policy

# Before running it, please change the variables between the 2 "========" lines
# They tell the script how many resources to create and other basical information(like subnet for lb creation).
# Each variable are well commented below.

# As we know, the lbaasv2 resources have a tree dependency, such like: a member belongs to a pool, a pool belongs to a loadbalancer.
# If a loadbalancer creation operation fails, this script will not stop, and continue to create pool, which will fail, of course.
# Use CTRL + C to stop the script if necessary, or
# Check the output(Execution Report) to see the 'Failed Command List:' and manually run them, so that
# The later batch operations are not effected.

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

# range list: tells the batch command binary how many resources to operate, and names.
# Abbreviations stand for:
# lb: loadbalancer
# pl: pool
# ls: listener
# mb: member
# hm: healthmonitor (only one healthmonitor can be assigned to a pool, so need for this variable definition)
# l7p: l7policy
lbrange=1-5
plrange=1,2
lsrange=80-85
mbrange=11-40
l7prange=1-2

# ====================================================

dts=`date +%Y-%m-%d-%H-%M-%S`
source $openrc

# create loadbalancer
$batchbin --output-filepath $output_dir/create_lb_$dts.json --check-lb lb-%{lbrange} \
    -- loadbalancer-create --name lb-%{lbrange} %{subnet} \
    ++ lbrange:$lbrange subnet:$subnet

# create pool
$batchbin --output-filepath $output_dir/create_pl_$dts.json --check-lb lb-%{lbrange} \
    -- pool-create --name pl-%{lbrange}-%{plrange} --lb-algorithm ROUND_ROBIN --loadbalancer lb-%{lbrange} --protocol HTTP \
    ++ lbrange:$lbrange plrange:$plrange

# create listener
$batchbin --output-filepath $output_dir/create_ls_$dts.json --check-lb lb-%{lbrange} \
    -- listener-create --name ls-%{lbrange}-%{lsrange} --default-pool pl-%{lbrange}-1 --loadbalancer lb-%{lbrange} \
        --protocol HTTP --protocol-port %{lsrange} \
    ++ lbrange:$lbrange lsrange:$lsrange

# create member
$batchbin --output-filepath $output_dir/create_mb_$dts.json --check-lb lb-%{lbrange} \
    -- member-create --name mb-%{lbrange}-%{plrange}-%{mbrange} --subnet %{subnet} \
        --address %{lbrange}.10.10.%{mbrange} --protocol-port 80 pl-%{lbrange}-%{plrange} \
    ++ lbrange:$lbrange plrange:$plrange mbrange:$mbrange subnet:$subnet

# create healthmonitor
$batchbin --output-filepath $output_dir/create_hm_$dts.json --check-lb lb-%{lbrange} \
    -- healthmonitor-create --name hm-%{lbrange}-%{plrange} \
        --timeout 15 --delay 15 --max-retries 5 --type PING --pool pl-%{lbrange}-%{plrange} \
    ++ lbrange:$lbrange plrange:$plrange

# create l7policy
$batchbin --output-filepath $output_dir/create_l7p_$dts.json --check-lb lb-%{lbrange} \
    -- l7policy-create --name l7p-%{lbrange}-%{lsrange}-%{l7prange} --action REJECT --listener ls-%{lbrange}-%{lsrange} \
    ++ lbrange:$lbrange lsrange:$lsrange l7prange:$l7prange

# The following examples show how to use <resource-type>-delete.
# If you want to delete ALL loadbalancer resources, use 'remove-lbaasv2-objects.sh' instead.

# $batchbin --output-filepath $output_dir/delete_l7p_$dts.json --check-lb lb-%{lbrange} -- l7policy-delete l7p-%{lbrange}-%{lsrange}-%{l7prange} \
#     ++ lbrange:$lbrange lsrange:$lsrange l7prange:$l7prange
# $batchbin --output-filepath $output_dir/delete_hm_$dts.json --check-lb lb-%{lbrange} -- healthmonitor-delete hm-%{lbrange}-%{plrange} ++ lbrange:$lbrange plrange:$plrange
# $batchbin --output-filepath $output_dir/delete_mb_$dts.json --check-lb lb-%{lbrange} -- member-delete mb-%{lbrange}-%{plrange}-%{mbrange} pl-%{lbrange}-%{plrange} \
#     ++ lbrange:$lbrange plrange:$plrange mbrange:$mbrange
# $batchbin --output-filepath $output_dir/delete_ls_$dts.json --check-lb lb-%{lbrange} -- listener-delete ls-%{lbrange}-%{lsrange} ++ lbrange:$lbrange lsrange:$lsrange
# $batchbin --output-filepath $output_dir/delete_pl_$dts.json --check-lb lb-%{lbrange} -- pool-delete pl-%{lbrange}-%{plrange} ++ lbrange:$lbrange plrange:$plrange
# $batchbin --output-filepath $output_dir/delete_lb_$dts.json --check-lb lb-%{lbrange} -- loadbalancer-delete lb-%{lbrange} ++ lbrange:$lbrange
