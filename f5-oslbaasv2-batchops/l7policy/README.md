L7policy Performance Test

## Testing Steps

1. Create resource.

    Execute `ansible-playbook -i env-kddi.ini -e @vars-model-5.yml create-resource-in-batch.yml -e index=0` to create 
    lb resource.

2. Run test model.

    We provide two testing models, you can execute `ansible-playbook -i env-kddi.ini l7policy/one-many-one.yml -e l7policy_num=100` 
    or `ansible-playbook -i env-kddi.ini l7policy/one-one-many.yml -e l7rule_num=100` to create l7policy and l7rule resources.

3. Parse log

    copy `server.log` and `f5-openstack-agent-CORE.log` from neutron hosts to `f5-oslbaasv2-parselog/logs`, then execute
    `./f5-oslbaasv2-parselog-linux-amd64 --logpath logs/server.log --logpath logs/f5-openstack-agent-CORE.log` to get
    `result.csv`.

## Tools

1. `ansible-playbook -i env-kddi.ini l7policy/delete-l7policy.yml -e l7policy_num=100` can delete the policy we create earlier.