#!/bin/bash

workdir=`cd $(dirname $0); pwd`
if [ $# -ne 1 ] || [ ! -f $1 ]; then
	echo "$0 <batchops.conf> or $1 not exists"
	exit 1
fi

$workdir/_create-projects.sh $1
$workdir/_create-lbaasv2-objects-layer-order.sh $1
