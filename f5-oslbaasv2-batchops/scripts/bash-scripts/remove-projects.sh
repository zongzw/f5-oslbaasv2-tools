#!/bin/bash

workdir=`cd $(dirname $0); pwd`
if [ $# -ne 1 ] || [ ! -f $1 ]; then
	echo "$0 <batchops.conf> or $1 not exists"
	exit 1
fi

source $1

source $openrc

for n in `openstack project list --format value --column Name | grep -E "$prefix_proj"`; do
    echo "deleting project $n ..."
    openstack project delete $n
done
