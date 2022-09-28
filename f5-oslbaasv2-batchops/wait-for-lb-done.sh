#!/bin/bash

if [ $# -ne 1 ]; then
    echo "<lbname>"
    exit 1
fi

timeout=140
delay=3

MYSQL_CMD="mysql -uneutron -h$NEUTRONDB_HOSTNAME -p$NEUTRONDB_PASSWORD $NEUTRONDB_DATABASE"

while [ $timeout -gt 0 ]; do
    timeout=$(($timeout - 1))
    count=`echo "select count(*) as count from lbaas_loadbalancers where name = '$1';" | $MYSQL_CMD | grep -v count`
    if [ $count -ge 2 ]; then
        echo "multiple lb named $1"
        exit 1
    fi

    status=`echo "select provisioning_status from lbaas_loadbalancers where name = '$1';" | $MYSQL_CMD | grep -v provisioning_status`
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

echo "loadbalancer: $1: $status, timeout, reset provisioning_status to ERROR, quit." 
echo "update lbaas_loadbalancers set provisioning_status = 'ERROR' where name = '$1';" | $MYSQL_CMD
exit 1
