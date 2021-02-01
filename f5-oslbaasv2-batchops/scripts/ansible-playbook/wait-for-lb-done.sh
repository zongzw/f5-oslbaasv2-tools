#!/bin/bash

if [ $# -ne 2 ]; then
    echo "<name> <dbpass>"
    exit 1
fi

timeout=300
delay=3

while [ $timeout -gt 0 ]; do
    timeout=$(($timeout - 1))
    count=`echo "select count(*) as count from lbaas_loadbalancers where name = '$1';" | mysql -uneutron -p$2 neutron | grep -v count`
    if [ $count -ge 2 ]; then
        echo "multiple lb named $1"
        exit 1
    fi

    status=`echo "select provisioning_status from lbaas_loadbalancers where name = '$1';" | mysql -uneutron -p$2 neutron | grep -v provisioning_status`
    if [ $? -ne 0 -o "$status" = "" ]; then
        echo "no loadbalancer named $1, quit"
        exit 1
    fi
    if [ "$status" = "ACTIVE" -o "$status" = "ERROR" ]; then
        echo "loadbalancer: $1: $status"
        exit 0
    else
        echo "loadbalancer: $1: $status, waiting"
    fi
    sleep $delay
done

echo "loadbalancer: $1: $status, timeout, quit." 
exit 1
