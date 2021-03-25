#!/bin/bash

set -x 
if [ $# -ne 5 ]; then
    echo "arguments: <loadbalancers|listeners|pools|members|healthmonitors> <name> <dbhost> <dbpass> <dbname>"
    exit 1
fi

count=`echo "select count(*) as count from lbaas_$1 where name = '$2';" | mysql -uneutron -h$3 -p$4 $5 | grep -v count`
if [ $? -ne 0 ]; then
    echo "failed to check count from db"
    exit 1
elif [ $count -eq 0 ]; then
    echo "false"
else
    echo "true"
fi
