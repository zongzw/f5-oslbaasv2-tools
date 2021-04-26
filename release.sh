#!/bin/bash

./build.sh

targetdir=f5-oslbaasv2-tools-release
mkdir $targetdir

# for n in f5-oslbaasv2-batchops f5-oslbaasv2-parselog f5-oslbaasv2-taillog; do
for n in f5-oslbaasv2-batchops f5-oslbaasv2-parselog; do
    for m in dist scripts; do
        if [ -d $n/$m ]; then
            cp -r $n/$m $targetdir
        fi
    done 
done

mkdir $targetdir/output