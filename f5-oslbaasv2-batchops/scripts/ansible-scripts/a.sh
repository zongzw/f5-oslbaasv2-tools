#!/bin/bash

for n in `seq $1 $2`; do

    ansible-playbook \
        -i env-lab.ini \
        -e @vars-model-5.yml \
        create-resources-in-batch.yml -e index=$n

done
