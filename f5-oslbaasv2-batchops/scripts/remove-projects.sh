#!/bin/bash

workdir=`cd $(dirname $0); pwd`
source $workdir/batchops.conf

source $openrc

for n in `openstack project list --format value --column Name | grep -E "$project_prefix"`; do
    echo "deleting project $n ..."
    openstack project delete $n
done
