
## Test Models

1. `model1.yml`: create `acl_group_nums` acl groups. Testing the performance of creating acl group.
2. `model2.yml`: create one acl group, then add `acl_rule_nums` rules into the acl group. Testing the performance of creating acl rule.
3. `model3.yml`: create one acl group, then bind it to `acl_listener_nums` listeners. Testing the performance of binding acl group to listener.

Variables of each model are specified in `model-xx-vars.yml`, `index` means create resources in which project.

## Test Steps

1. Run `ansible-playbook -i env-kddi.ini -e @vars-model-5.yml create-resource-in-batch.yml -e index=0` to create lb resources.

2. Run `ansible-playbook -i env-kddi.ini -e @acl/model1-vars.yml acl/model1.yml` to test model1.

3. Collect log from every neutron server and parse them.