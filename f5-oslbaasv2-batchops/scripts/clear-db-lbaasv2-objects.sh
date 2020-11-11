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

echo "Deleting members $prefix_mb..."
echo "delete from lbaas_members where name like '$prefix_mb%';" | $pipe_suffix
echo "Deleting l7policies $prefix_l7p..."
echo "delete from lbaas_l7policies where name like '$prefix_l7p%';" | $pipe_suffix
echo "Deleting listeners $prefix_ls..."
echo "delete from lbaas_listeners where name like '$prefix_ls%';" | $pipe_suffix
echo "Deleting pools $prefix_pl..."
echo "delete from lbaas_pools where name like '$prefix_pl%';" | $pipe_suffix
echo "Deleting healthmonitors $prefix_hm..."
echo "delete from lbaas_healthmonitors where name like '$prefix_hm%';" | $pipe_suffix
echo "Deleting loadbalancers $prefix_lb..."
echo "delete from lbaas_loadbalancer_statistics where loadbalancer_id in \
	(select id from lbaas_loadbalancers where name like '$prefix_lb%');" | $pipe_suffix
echo "delete from lbaas_loadbalancers where name like '$prefix_lb%';" | $pipe_suffix

echo
echo "	F5 Agent will do purging on orphan lbaasv2 objects later."
echo "	Wait a moment and check lbaasv2 objects are removed from BIG-IP devices."
echo 
