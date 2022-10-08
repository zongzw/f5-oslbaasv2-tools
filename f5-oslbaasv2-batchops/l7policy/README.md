L7policy Performance Test

## Test Models

1. `one-many-one.yml`: one listener have many l7policies, every l7policy has one l7rule, testing the performance of creating l7policy int one listener.

2. `one-one-many.yml`: one listener have one l7policy, one l7policy has many l7rules, testing the performance of creating many l7rules in one l7policy.

## Test Steps

1. Create resource.

    Execute `ansible-playbook -i env-lab.ini -e @vars-model-5.yml create-resource-in-batch.yml -e index=0` to create 
    lb resource.

2. Run test model.

    We provide two testing models, you can execute `ansible-playbook -i env-lab.ini -e @l7policy/one-many-one-vars.yml l7policy/one-many-one.yml` 
    or `ansible-playbook -i env-lab.ini -e @l7policy/one-one-many-vars.yml l7policy/one-one-many.yml` to create l7policy and l7rule resources.

3. Parse log

    copy `server.log` and `f5-openstack-agent-CORE.log` from neutron hosts to `f5-oslbaasv2-parselog/logs`, then execute
    `./f5-oslbaasv2-parselog-linux-amd64 --logpath logs/server.log --logpath logs/f5-openstack-agent-CORE.log` to get
    `result.csv`.

## Tools

1. `ansible-playbook -i env-lab.ini l7policy/delete-l7policy.yml -e l7policy_num=100` can delete the policy we create earlier.