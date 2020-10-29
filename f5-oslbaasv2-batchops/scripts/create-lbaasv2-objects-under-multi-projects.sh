#!/bin/bash

workdir=`cd $(dirname $0); pwd`
source $workdir/batchops.conf

source $openrc

$workdir/create-projects.sh
$workdir/create-lbaasv2-objects.sh
