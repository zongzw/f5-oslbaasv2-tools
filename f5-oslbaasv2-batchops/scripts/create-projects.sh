#!/bin/bash

# This script is use to create multiple projects in OpenStack Project

workdir=`cd $(dirname $0); pwd`
if [ $# -ne 1 ] || [ ! -f $1 ]; then
	echo "$0 <batchops.conf> or $1 not exists"
	exit 1
fi

source $1

which openstack > /dev/null
if [ $? -ne 0 ]; then
    echo "No 'openstack' command line found, quit."
    exit 1
fi

which neutron > /dev/null
if [ $? -ne 0 ]; then
    echo "No 'neutron' command line found, quit."
    exit 1
fi

if [ ! -f $openrc ]; then
    echo "Invalid rc file: $openrc, quit."
    exit 1
fi
source $openrc

# For testing environment, we recommend to set the auota to unlimited.
function unlimit_lbaas_quota() {
    proj=$1

    neutron --os-project-name $proj quota-update --loadbalancer -1
    neutron --os-project-name $proj quota-update --healthmonitor -1
    neutron --os-project-name $proj quota-update --l7policy -1
    neutron --os-project-name $proj quota-update --listener -1
    neutron --os-project-name $proj quota-update --loadbalancer -1
    neutron --os-project-name $proj quota-update --member -1
    neutron --os-project-name $proj quota-update --pool -1

    # neutron --os-project-name $proj quota-update --network -1
    # neutron --os-project-name $proj quota-update --floatingip -1
    # neutron --os-project-name $proj quota-update --port -1
    # neutron --os-project-name $proj quota-update --router -1
    # neutron --os-project-name $proj quota-update --security_group -1
    # neutron --os-project-name $proj quota-update --subnet -1
}

index=$project_start_no
while [ $index -le $project_end_no ]; do
    project_name=$prefix_proj$index
    echo "Creating project: $project_name ..."
    #openstack project create --domain default $project_name 2> /dev/null
    openstack project create $project_name 2> /dev/null

    echo "Add user admin to project $project_name ..."
    openstack role add --project $project_name --user admin admin

    echo "Unlimit resources' quota ..."
    unlimit_lbaas_quota $project_name > /dev/null 2>&1

    index=$(($index + 1))
done
