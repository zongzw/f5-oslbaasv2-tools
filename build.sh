#!/bin/bash

cdir=`cd $(dirname $0); pwd`
progs="f5-oslbaasv2-batchops f5-oslbaasv2-parselog"

for n in $progs; do
    docker run --rm \
        -v "$cdir/$n":/usr/src/$n \
        -w /usr/src/$n golang:latest \
        bash -c '
            mkdir -p /usr/src/'$n'/dist
            rm -rf /usr/src/'$n'/dist
            export GOPROXY=https://goproxy.cn
            for GOOS in darwin linux; do
                for GOARCH in amd64; do
                    export GOOS GOARCH
                    echo '$n'-$GOOS-$GOARCH ...
                    go build -o 'dist/$n'-$GOOS-$GOARCH
                done
            done'
done


# Other platform and architecture if need.
# for GOOS in darwin linux windows; do
#     for GOARCH in 386 amd64; do