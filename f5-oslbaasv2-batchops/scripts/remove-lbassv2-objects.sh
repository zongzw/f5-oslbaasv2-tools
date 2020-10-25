#!/bin/bash

echo "Deleting healthmonitor ..."
hms=`neutron lbaas-healthmonitor-list -f value -c id`
for j in $hms; do
    echo "Deleting healthmonitor $j"
    neutron lbaas-healthmonitor-delete $j
    sleep 1
    echo 
done

echo "Deleting pool and members ..."
pls=`neutron lbaas-pool-list -f value -c id`
for l in $pls; do
    mbs=`neutron lbaas-member-list -f value -c id $l`
    for i in $mbs; do 
        echo "Deleting $i from $l ..."
	neutron lbaas-member-delete $i $l
        sleep 1
        echo 
    done
    echo "Deleting pool $l ..."
    neutron lbaas-pool-delete $l
    sleep 1
    echo 
done

echo "Deleting l7policies ..."
l7ps=`neutron lbaas-l7policy-list -f value -c id`
for l in $l7ps; do
    echo "Deleting l7policy $l ..."
    neutron lbaas-l7policy-delete $l
    sleep 1
    echo
done

echo "Deleting listeners ..."
lss=`neutron lbaas-listener-list -f value -c id`
for m in $lss; do 
    echo "Deleting listener $m ..."
    neutron lbaas-listener-delete $m
    sleep 1
    echo 
done

echo "Deleting loadbalancers ..."
lbs=`neutron lbaas-loadbalancer-list -f value -c id`
for n in $lbs; do 
    echo "Deleting loadbalancer $n ..."
    neutron lbaas-loadbalancer-delete $n
    sleep 1
    echo
done
