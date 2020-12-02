#!/bin/bash

if [ $# -ne 1 ]; then
	echo "$0 <partition_no>"
	exit 1
fi
workdir=`cd $(dirname $0); pwd`
n=$1

PROJECT_START_NO=$n PROJECT_END_NO=$n $workdir/create-projects.sh 
( PROJECT_START_NO=$n PROJECT_END_NO=$n $workdir/create-lbaasv2-objects-layer-order.sh > $workdir/../output/$n.log & )
