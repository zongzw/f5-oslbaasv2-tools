#!/bin/bash

workdir=`cd $(dirname $0); pwd`
source $workdir/batchops.conf

which mysql
if [ $? -ne 0 ]; then
    echo "canot find mysql client."
    exit 1
fi

pipe_suffix="mysql -u$neutron_db_username -p$neutron_db_password -h$neutron_db_host $neutron_db_name"

#echo "select id, name, provisioning_status from lbaas_loadbalancers;" | $pipe_suffix

echo "Listing members $prefix_mb..."
echo "select id, name, provisioning_status from lbaas_members where name like '$prefix_mb%';" | $pipe_suffix
echo "Listing l7policies $prefix_l7p..."
echo "select id, name, provisioning_status from lbaas_l7policies where name like '$prefix_l7p%';" | $pipe_suffix
echo "Listing listeners $prefix_ls..."
echo "select id, name, provisioning_status from lbaas_listeners where name like '$prefix_ls%';" | $pipe_suffix
echo "Listing pools $prefix_pl..."
echo "select id, name, provisioning_status from lbaas_pools where name like '$prefix_pl%';" | $pipe_suffix
echo "Listing healthmonitors $prefix_hm..."
echo "select id, name, provisioning_status from lbaas_healthmonitors where name like '$prefix_hm%';" | $pipe_suffix
echo "Listing loadbalancers $prefix_lb..."
echo "select id, name, provisioning_status from lbaas_loadbalancers where name like '$prefix_lb%';" | $pipe_suffix

