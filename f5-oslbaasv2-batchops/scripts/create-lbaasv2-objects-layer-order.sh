#!/bin/bash

# This is a performance test script.
# In sequence, it will create loadbalancer > pool > listener > member > healthmonitor > l7policy
# Different with -tree-order, this script creates all LB, then creates all pools, next all listener ... all members.

# Before running it, please change the variables between the 2 "========" lines in batchops.conf
# They tell the script how many resources to create and other basical information(like subnet for lb creation).
# Each variable are well commented below.

# As we know, the lbaasv2 resources have a tree dependency, such like: a member belongs to a pool, a pool belongs to a loadbalancer.
# If a loadbalancer creation operation fails, this script will not stop, and continue to create pool, which will fail, of course.
# Use CTRL + C to stop the script if necessary, or
# Check the output(Execution Report) to see the 'Failed Command List:' and manually run them, so that
# The later batch operations are not effected.

workdir=`cd $(dirname $0); pwd`
source $workdir/batchops.conf

dts=`date +%Y-%m-%d-%H-%M-%S`
source $openrc

# create loadbalancer
$batchbin --output-filepath $output_dir/create_lb_$dts.json --check-lb lb-%{pjrange}-%{lbrange} \
    -- --os-project-name proj_%{pjrange} lbaas-loadbalancer-create --name lb-%{pjrange}-%{lbrange} %{subnet} \
    ++ pjrange:$pjrange lbrange:$lbrange subnet:$subnet

# create pool
$batchbin --output-filepath $output_dir/create_pl_$dts.json --check-lb lb-%{pjrange}-%{lbrange} \
    -- --os-project-name proj_%{pjrange} lbaas-pool-create --name pl-%{pjrange}-%{lbrange}-%{plrange} \
        --lb-algorithm ROUND_ROBIN --loadbalancer lb-%{pjrange}-%{lbrange} --protocol HTTP \
    ++ pjrange:$pjrange lbrange:$lbrange plrange:$plrange

# create listener
$batchbin --output-filepath $output_dir/create_ls_$dts.json --check-lb lb-%{pjrange}-%{lbrange} \
    -- --os-project-name proj_%{pjrange} lbaas-listener-create --name ls-%{pjrange}-%{lbrange}-%{lsrange} \
        --default-pool pl-%{pjrange}-%{lbrange}-1 --loadbalancer lb-%{pjrange}-%{lbrange} --protocol HTTP --protocol-port %{lsrange} \
    ++ pjrange:$pjrange lbrange:$lbrange lsrange:$lsrange

# create member
$batchbin --output-filepath $output_dir/create_mb_$dts.json --check-lb lb-%{pjrange}-%{lbrange} \
    -- --os-project-name proj_%{pjrange} lbaas-member-create --name mb-%{pjrange}-%{lbrange}-%{plrange}-%{mbrange} \
        --subnet %{subnet} --address %{pjrange}.%{lbrange}.%{plrange}.%{mbrange} --protocol-port 80 pl-%{pjrange}-%{lbrange}-%{plrange} \
    ++ pjrange:$pjrange lbrange:$lbrange plrange:$plrange mbrange:$mbrange subnet:$subnet

# create healthmonitor
$batchbin --output-filepath $output_dir/create_hm_$dts.json --check-lb lb-%{pjrange}-%{lbrange} \
    -- --os-project-name proj_%{pjrange} lbaas-healthmonitor-create --name hm-%{pjrange}-%{lbrange}-%{plrange} \
        --timeout 15 --delay 15 --max-retries 5 --type PING --pool pl-%{pjrange}-%{lbrange}-%{plrange} \
    ++ pjrange:$pjrange lbrange:$lbrange plrange:$plrange

# create l7policy
$batchbin --output-filepath $output_dir/create_l7p_$dts.json --check-lb lb-%{pjrange}-%{lbrange} \
    -- --os-project-name proj_%{pjrange} lbaas-l7policy-create --name l7p-%{pjrange}-%{lbrange}-%{lsrange}-%{l7prange} \
        --action REJECT --listener ls-%{pjrange}-%{lbrange}-%{lsrange} \
    ++ pjrange:$pjrange lbrange:$lbrange lsrange:$lsrange l7prange:$l7prange

# The following examples show how to use <resource-type>-delete.
# If you want to delete ALL loadbalancer resources, use 'remove-lbaasv2-objects.sh' instead.

# $batchbin --output-filepath $output_dir/delete_l7p_$dts.json --check-lb lb-%{pjrange}-%{lbrange} -- l7policy-delete l7p-%{pjrange}-%{lbrange}-%{lsrange}-%{l7prange} \
#     ++ lbrange:$lbrange lsrange:$lsrange l7prange:$l7prange
# $batchbin --output-filepath $output_dir/delete_hm_$dts.json --check-lb lb-%{pjrange}-%{lbrange} -- healthmonitor-delete hm-%{pjrange}-%{lbrange}-%{plrange} ++ lbrange:$lbrange plrange:$plrange
# $batchbin --output-filepath $output_dir/delete_mb_$dts.json --check-lb lb-%{pjrange}-%{lbrange} -- member-delete mb-%{pjrange}-%{lbrange}-%{plrange}-%{mbrange} pl-%{pjrange}-%{lbrange}-%{plrange} \
#     ++ lbrange:$lbrange plrange:$plrange mbrange:$mbrange
# $batchbin --output-filepath $output_dir/delete_ls_$dts.json --check-lb lb-%{pjrange}-%{lbrange} -- listener-delete ls-%{pjrange}-%{lbrange}-%{lsrange} ++ lbrange:$lbrange lsrange:$lsrange
# $batchbin --output-filepath $output_dir/delete_pl_$dts.json --check-lb lb-%{pjrange}-%{lbrange} -- pool-delete pl-%{pjrange}-%{lbrange}-%{plrange} ++ lbrange:$lbrange plrange:$plrange
# $batchbin --output-filepath $output_dir/delete_lb_$dts.json --check-lb lb-%{pjrange}-%{lbrange} -- loadbalancer-delete lb-%{pjrange}-%{lbrange} ++ lbrange:$lbrange
