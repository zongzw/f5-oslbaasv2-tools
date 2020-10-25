#!/bin/bash

cdir=`cd $(dirname $0); pwd`
openrc=$cdir/openrc
subnet=private-subnet

lbrange=1-2
plrange=1
lsrange=80-81
mbrange=11-12
l7prange=1-2

batchbin=$cdir/../dist/f5-oslbaasv2-batchops-darwin-amd64

dts=`date +%Y-%m-%d-%H-%M-%S`
output_dir=$cdir/../output

$batchbin --output-filepath $output_dir/create_lb_$dts.json --check-lb lb-%{lbrange} \
    -- loadbalancer-create --name lb-%{lbrange} %{subnet} \
    ++ lbrange:$lbrange subnet:$subnet

$batchbin --output-filepath $output_dir/create_pl_$dts.json --check-lb lb-%{lbrange} \
    -- pool-create --name pl-%{lbrange}-%{plrange} --lb-algorithm ROUND_ROBIN --loadbalancer lb-%{lbrange} --protocol HTTP \
    ++ lbrange:$lbrange plrange:$plrange

$batchbin --output-filepath $output_dir/create_ls_$dts.json --check-lb lb-%{lbrange} \
    -- listener-create --name ls-%{lbrange}-%{lsrange} --default-pool pl-%{lbrange}-1 --loadbalancer lb-%{lbrange} \
        --protocol HTTP --protocol-port %{lsrange} \
    ++ lbrange:$lbrange lsrange:$lsrange

$batchbin --output-filepath $output_dir/create_mb_$dts.json --check-lb lb-%{lbrange} \
    -- member-create --name mb-%{lbrange}-%{plrange}-%{mbrange} --subnet %{subnet} \
        --address %{lbrange}.10.10.%{mbrange} --protocol-port 80 pl-%{lbrange}-%{plrange} \
    ++ lbrange:$lbrange plrange:$plrange mbrange:$mbrange subnet:$subnet

$batchbin --output-filepath $output_dir/create_hm_$dts.json --check-lb lb-%{lbrange} \
    -- healthmonitor-create --name hm-%{lbrange}-%{plrange} \
        --timeout 15 --delay 15 --max-retries 5 --type PING --pool pl-%{lbrange}-%{plrange} \
    ++ lbrange:$lbrange plrange:$plrange

$batchbin --output-filepath $output_dir/create_l7p_$dts.json --check-lb lb-%{lbrange} \
    -- l7policy-create --name l7p-%{lbrange}-%{lsrange}-%{l7prange} --action REJECT --listener ls-%{lbrange}-%{lsrange} \
    ++ lbrange:$lbrange lsrange:$lsrange l7prange:$l7prange

# $batchbin --output-filepath $output_dir/delete_l7p_$dts.json --check-lb lb-%{lbrange} -- l7policy-delete l7p-%{lbrange}-%{lsrange}-%{l7prange} \
#     ++ lbrange:$lbrange lsrange:$lsrange l7prange:$l7prange
# $batchbin --output-filepath $output_dir/delete_hm_$dts.json --check-lb lb-%{lbrange} -- healthmonitor-delete hm-%{lbrange}-%{plrange} ++ lbrange:$lbrange plrange:$plrange
# $batchbin --output-filepath $output_dir/delete_mb_$dts.json --check-lb lb-%{lbrange} -- member-delete mb-%{lbrange}-%{plrange}-%{mbrange} pl-%{lbrange}-%{plrange} \
#     ++ lbrange:$lbrange plrange:$plrange mbrange:$mbrange
# $batchbin --output-filepath $output_dir/delete_ls_$dts.json --check-lb lb-%{lbrange} -- listener-delete ls-%{lbrange}-%{lsrange} ++ lbrange:$lbrange lsrange:$lsrange
# $batchbin --output-filepath $output_dir/delete_pl_$dts.json --check-lb lb-%{lbrange} -- pool-delete pl-%{lbrange}-%{plrange} ++ lbrange:$lbrange plrange:$plrange
# $batchbin --output-filepath $output_dir/delete_lb_$dts.json --check-lb lb-%{lbrange} -- loadbalancer-delete lb-%{lbrange} ++ lbrange:$lbrange
