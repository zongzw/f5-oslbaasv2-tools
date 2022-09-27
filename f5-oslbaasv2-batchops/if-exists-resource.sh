#!/bin/bash

set -x 
if [ $# -ne 2 ]; then
    echo "arguments: <loadbalancers|listeners|pools|members|healthmonitors|l7policies> <name>"
    exit 1
fi

MYSQL_CMD="mysql -uneutron -h$NEUTRONDB_HOSTNAME -p$NEUTRONDB_PASSWORD $NEUTRONDB_DATABASE"

count=`echo "select count(*) as count from lbaas_$1 where name = '$2';" | $MYSQL_CMD | grep -v count`
if [ $? -ne 0 ]; then
    echo "failed to check count from db"
    exit 1
elif [ $count -eq 0 ]; then
    echo "false"
else
    echo "true"
fi
