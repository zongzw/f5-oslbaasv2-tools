# f5-oslbaasv2-batchops

This is a program(in golang) to help running `neutron` commands in batch.
It is usually used to create dozens of loadbalancer resources, such as loadbalancer/listener/pool/member/healthmonitor/l7policy/l7rule for performance test.

## Usage

The command arguments includes 3 parts, as shown in below 'help and example':

* **\[command arguments]**: The arguments to control the command's behavior, like `--output <filepath>` tells the command where to save the result.
* **\<neutron command>**: The neutron command template(with variables declared inside if any). Its syntex complies with native neutron commands(`neutron lbaas-*`) but 
  * For ease of use, the prefix `neutron lbaas-` should NOT be included here. For example, `neutron lbaas-loadbalancer-list`, change to `loadbalancer-list` instead here.
  * Use `%{<variable-name>}` to indicate `neutron command` is a command template which would be executed multiple times according to variable‘s value.
* **\[variable-definition]**: Corresponding to `variable-name`, `variable-definition` tells the values used in the command template. The format of `variable-definition` is composed of `variable-name` and `values`. The `values` can be number range joint with `-` or string enumeration joint with `,`. For example:
  * `1-5`: [1 2 3 4 5]
  * `1-5,7,8,a,b,c`: [1 2 3 4 5 7 8 a b c]
  * `private-subnet,public-subnet`: [private-subnet public-subnet]

These 3 parts are divided with `--` and `++` as shown below.

### Help and Example

```
# ./dist/f5-oslbaasv2-batchops-linux-amd64 --help
Usage: 

    ./dist/f5-oslbaasv2-batchops-linux-amd64 [command arguments] -- <neutron command and arguments>[ ++ variable-definition]

Example:

    ./dist/f5-oslbaasv2-batchops-linux-amd64 --output /dev/stdout \
    -- loadbalancer-create --name lb{x} {y} \
    ++ x:1-5 y:private-subnet,public-subnet

Command Arguments: 

  -output string
        output the result (default "/dev/stdout")

```

### Output and logs

Run command as:

```
# ./dist/f5-oslbaasv2-batchops-linux-amd64 --output rlt.json -- loadbalancer-show lb%{x} ++ x:1-5
```
logging:

```
2020/10/20 18:25:46 output to: rlt.json
2020/10/20 18:25:46 Command template: loadbalancer-show lb%{x}
2020/10/20 18:25:46 found variable: %{x}
2020/10/20 18:25:46 variables parsed as
2020/10/20 18:25:46          x: [1 2 3 4 5]
2020/10/20 18:25:46 neutron command: /usr/local/bin/neutron
2020/10/20 18:25:46 Command(1/5): 'neutron lbaas-loadbalancer-show lb1' starts
2020/10/20 18:25:54 Command(1/5): exits with: 0, executing time: 7.863804828s
2020/10/20 18:25:54 Command(1/5): Checking Execution
2020/10/20 18:25:54 Command(1/5): check done, done time: 7.863806017s
2020/10/20 18:25:54 Command(2/5): 'neutron lbaas-loadbalancer-show lb2' starts
2020/10/20 18:26:00 Command(2/5): exits with: 0, executing time: 6.82681959s
2020/10/20 18:26:00 Command(2/5): Checking Execution
2020/10/20 18:26:00 Command(2/5): check done, done time: 6.826820846s
2020/10/20 18:26:00 Command(3/5): 'neutron lbaas-loadbalancer-show lb3' starts
2020/10/20 18:26:06 Command(3/5): exits with: 0, executing time: 5.751434801s
2020/10/20 18:26:06 Command(3/5): Checking Execution
2020/10/20 18:26:06 Command(3/5): check done, done time: 5.751436141s
2020/10/20 18:26:06 Command(4/5): 'neutron lbaas-loadbalancer-show lb4' starts
2020/10/20 18:26:12 Command(4/5): exits with: 0, executing time: 5.489224268s
2020/10/20 18:26:12 Command(4/5): Checking Execution
2020/10/20 18:26:12 Command(4/5): check done, done time: 5.48922505s
2020/10/20 18:26:12 Command(5/5): 'neutron lbaas-loadbalancer-show lb5' starts
2020/10/20 18:26:16 Command(5/5): exits with: 0, executing time: 4.582686565s
2020/10/20 18:26:16 Command(5/5): Checking Execution
2020/10/20 18:26:16 Command(5/5): check done, done time: 4.582688698s
2020/10/20 18:26:16 Writen to file rlt.json: data-len:4976
```

rlt.json

```
[
  {
    "seqnum": 1,
    "command": "neutron lbaas-loadbalancer-show lb1",
    "output": {
      "admin_state_up": true,
      "description": "",
      "id": "0afad36a-337c-4298-8649-fc8c228b0f91",
      "listeners": [
        {
          "id": "6a9ac705-d7da-4214-b99f-fc47fbf760c8"
        }
      ],
      "name": "lb1",
      "operating_status": "ONLINE",
      "pools": [
        {
          "id": "6adfc3b9-33e4-446b-99ea-0eae873140e1"
        }
      ],
      "provider": "f5networks",
      "provisioning_status": "ACTIVE",
      "tenant_id": "38ac07a46dad448cb93bec736ba89f1c",
      "vip_address": "10.0.0.31",
      "vip_port_id": "e3a9262e-5ca1-4cf2-9e7c-fe6191cb1be2",
      "vip_subnet_id": "b6fc5c77-d727-456e-bbd8-67a82534676c"
    },
    "error": "neutron CLI is deprecated and will be removed in the future. Use openstack CLI instead.\n",
    "exitcode": 0,
    "cmd_duration": 7863804828,
    "success": "lbaas-loadbalancer-show done",
    "done_duration": 7863806017
  },
  ...
  {
    "seqnum": 5,
    "command": "neutron lbaas-loadbalancer-show lb5",
    "output": {
      "admin_state_up": true,
    ...
    "cmd_duration": 4582686565,
    "success": "lbaas-loadbalancer-show done",
    "done_duration": 4582688698
  }
]
```