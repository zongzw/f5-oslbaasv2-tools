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

if [ $# -ne 1 ] || [ ! -f $1 ]; then
	echo "$0 <batchops.conf> or $1 not exists"
	exit 1
fi

source $1
source $openrc

dts=`date +%Y-%m-%d-%H-%M-%S`

# create loadbalancer
$batchbin --max-check-times 1024 \
      --mysql-uri $neutron_db_username:$neutron_db_password@tcp($neutron_db_host:3306)/$neutron_db_host \
      --output-filepath $output_dir/create_lb_$dts.json \
      --check-lb $prefix_lb%{pjrange}-%{lbrange} \
    -- --os-project-name $prefix_proj%{pjrange} lbaas-loadbalancer-create --provider $provider \
       --name $prefix_lb%{pjrange}-%{lbrange} %{subnet} \
    ++ pjrange:$pjrange lbrange:$lbrange subnet:$subnet

# create pool
$batchbin --max-check-times 1024 \ 
      --mysql-uri $neutron_db_username:$neutron_db_password@tcp($neutron_db_host:3306)/$neutron_db_host \
      --output-filepath $output_dir/create_pl_$dts.json \
      --check-lb $prefix_lb%{pjrange}-%{lbrange} \
    -- --os-project-name $prefix_proj%{pjrange} lbaas-pool-create --name $prefix_pl%{pjrange}-%{lbrange}-%{plrange} \
        --lb-algorithm ROUND_ROBIN --loadbalancer $prefix_lb%{pjrange}-%{lbrange} --protocol HTTP \
    ++ pjrange:$pjrange lbrange:$lbrange plrange:$plrange

# create healthmonitor
$batchbin --max-check-times 1024 \ 
      --mysql-uri $neutron_db_username:$neutron_db_password@tcp($neutron_db_host:3306)/$neutron_db_host \
      --output-filepath $output_dir/create_hm_$dts.json \
      --check-lb $prefix_lb%{pjrange}-%{lbrange} \
    -- --os-project-name $prefix_proj%{pjrange} lbaas-healthmonitor-create --name $prefix_hm%{pjrange}-%{lbrange}-%{plrange} \
        --timeout 15 --delay 15 --max-retries 5 --type PING --pool $prefix_pl%{pjrange}-%{lbrange}-%{plrange} \
    ++ pjrange:$pjrange lbrange:$lbrange plrange:$plrange

# create listener
$batchbin --max-check-times 1024 \ 
      --mysql-uri $neutron_db_username:$neutron_db_password@tcp($neutron_db_host:3306)/$neutron_db_host \
      --output-filepath $output_dir/create_ls_$dts.json \
      --check-lb $prefix_lb%{pjrange}-%{lbrange} \
    -- --os-project-name $prefix_proj%{pjrange} lbaas-listener-create --name $prefix_ls%{pjrange}-%{lbrange}-%{lsrange} \
       --default-pool $prefix_pl%{pjrange}-%{lbrange}-1 --loadbalancer $prefix_lb%{pjrange}-%{lbrange} \
       --protocol HTTP --protocol-port %{lsrange} \
    ++ pjrange:$pjrange lbrange:$lbrange lsrange:$lsrange

# create member
$batchbin --max-check-times 1024 \ 
      --mysql-uri $neutron_db_username:$neutron_db_password@tcp($neutron_db_host:3306)/$neutron_db_host \
      --output-filepath $output_dir/create_mb_$dts.json \
      --check-lb $prefix_lb%{pjrange}-%{lbrange} \
    -- --os-project-name $prefix_proj%{pjrange} lbaas-member-create --name $prefix_mb%{pjrange}-%{lbrange}-%{plrange}-%{mbrange} \
       --subnet %{subnet} --address %{pjrange}.%{lbrange}.%{plrange}.%{mbrange} --protocol-port 80 \
       $prefix_pl%{pjrange}-%{lbrange}-%{plrange} \
    ++ pjrange:$pjrange lbrange:$lbrange plrange:$plrange mbrange:$mbrange subnet:$subnet

# create l7policy
$batchbin --max-check-times 1024 \ 
      --mysql-uri $neutron_db_username:$neutron_db_password@tcp($neutron_db_host:3306)/$neutron_db_host \
      --output-filepath $output_dir/create_l7p_$dts.json \
      --check-lb $prefix_lb%{pjrange}-%{lbrange} \
    -- --os-project-name $prefix_proj%{pjrange} lbaas-l7policy-create \
       --name $prefix_l7p%{pjrange}-%{lbrange}-%{lsrange}-%{l7prange} \
       --action REJECT --listener $prefix_ls%{pjrange}-%{lbrange}-%{lsrange} \
    ++ pjrange:$pjrange lbrange:$lbrange lsrange:$lsrange l7prange:$l7prange

