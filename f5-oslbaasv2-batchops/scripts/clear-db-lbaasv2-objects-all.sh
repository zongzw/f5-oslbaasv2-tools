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

echo "Deleting members ..."
echo "delete from lbaas_members;" | $pipe_suffix
echo "Deleting l7policies ..."
echo "delete from lbaas_l7policies;" | $pipe_suffix
echo "Deleting sni ..."
echo "delete from lbaas_sni;" | $pipe_suffix
echo "Deleting listeners ..."
echo "delete from lbaas_listeners;" | $pipe_suffix
echo "Deleting sessionpersistence ..."
echo "delete from lbaas_sessionpersistences;" | $pipe_suffix
echo "Deleting pools ..."
echo "delete from lbaas_pools;" | $pipe_suffix
echo "Deleting healthmonitors ..."
echo "delete from lbaas_healthmonitors;" | $pipe_suffix
echo "Deleting loadbalancers ..."
echo "delete from lbaas_loadbalancer_statistics ;" | $pipe_suffix
echo "delete from lbaas_loadbalancers;" | $pipe_suffix

echo
echo "	F5 Agent will do purging on orphan lbaasv2 objects later."
echo "	Wait a moment and check lbaasv2 objects are removed from BIG-IP devices."
echo 
