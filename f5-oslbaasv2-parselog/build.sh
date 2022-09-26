#!/bin/bash

cdir=`cd $(dirname $0); pwd`
prog="f5-oslbaasv2-parselog"

docker run --rm \
    -v "$cdir":/usr/src/$prog \
    --env prog=$prog \
    -w /usr/src/$prog golang:latest \
    bash -c '
        mkdir -p /usr/src/$prog/dist
        rm -rf /usr/src/$prog/dist
        echo nameserver 114.114.114.114 > /etc/resolv.conf
        export GOPROXY=https://goproxy.cn
        for GOOS in darwin linux; do
        #for GOOS in linux; do
            for GOARCH in amd64; do
                export GOOS GOARCH
                echo $prog-$GOOS-$GOARCH ...
                go build -o dist/$prog-$GOOS-$GOARCH
            done
        done'


# Other platform and architecture if need.
# for GOOS in darwin linux windows; do
#     for GOARCH in 386 amd64; do
