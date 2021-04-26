#!/bin/bash

cdir=`cd $(dirname $0); pwd`
$cdir/build.sh

targetdir=$cdir/f5-oslbaasv2-tools-release
mkdir -p $targetdir

cp -r $cdir/f5-oslbaasv2-batchops/scripts/ansible-scripts $targetdir/f5-oslbaasv2-batchops
cp -r $cdir/f5-oslbaasv2-parselog/dist/f5-oslbaasv2-parselog-linux-amd64 $targetdir

mkdir -p $targetdir/output