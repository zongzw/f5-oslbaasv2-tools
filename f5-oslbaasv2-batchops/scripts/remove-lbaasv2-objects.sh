#!/bin/bash

# This is the script to delete ALL lbaasv2 resources, So BE CAUTIOUS to use this script. 
# If you have lbaasv2 resources rather than performance test, DON'T use this script.

# *************** BE CAUTIOUS, THIS SCRIPT WILL REMOVE ALL LBAASV2 OBJECTS ***************

# In sequence, it will delete all healthmonitor > pool and member > l7policy > listener > loadbalancer.
# The script use 'neutron lbaas-<resource_type>-list -f value -c id' to find resource id, 
# and then use 'neutron lbaas-<resource_type>-delete' to delete them one by one.

# healthmonitor

workdir=`cd $(dirname $0); pwd`
source $workdir/batchops.conf

source $openrc

echo "Deleting healthmonitor ..."
hms=`neutron lbaas-healthmonitor-list -f value -c id`
for j in $hms; do
    echo "Deleting healthmonitor $j"
    neutron lbaas-healthmonitor-delete $j
    sleep 1
    echo 
done

# pool and member
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

# l7policy
echo "Deleting l7policies ..."
l7ps=`neutron lbaas-l7policy-list -f value -c id`
for l in $l7ps; do
    echo "Deleting l7policy $l ..."
    neutron lbaas-l7policy-delete $l
    sleep 1
    echo
done

# listener
echo "Deleting listeners ..."
lss=`neutron lbaas-listener-list -f value -c id`
for m in $lss; do 
    echo "Deleting listener $m ..."
    neutron lbaas-listener-delete $m
    sleep 1
    echo 
done

# loadbalancer
echo "Deleting loadbalancers ..."
lbs=`neutron lbaas-loadbalancer-list -f value -c id`
for n in $lbs; do 
    echo "Deleting loadbalancer $n ..."
    neutron lbaas-loadbalancer-delete $n
    sleep 1
    echo
done
