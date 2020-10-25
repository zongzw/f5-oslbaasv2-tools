# f5-oslbaasv2-batchops

This is a program(in golang) to help running `neutron` commands in batch.
It is usually used to create dozens of loadbalancer resources, such as loadbalancer/listener/pool/member/healthmonitor/l7policy/l7rule for performance test.

## Usage

The command arguments includes 3 parts, as shown in below 'help and example':

* **\[command arguments]**: The arguments to control the command's behavior, like `--output-filepath <filepath>` tells the command where to save the result; `--check-lb <lb id or name>` tells the command which loadbalancer this operated resource belongs to(for check purpose).
* **\<neutron command>**: The neutron command template(with variables declared inside if any). Its syntex complies with native neutron commands(`neutron lbaas-*`) but 
  * For ease of use, the prefix `neutron lbaas-` should NOT be included here. For example, `neutron lbaas-loadbalancer-list`, change to `loadbalancer-list` instead here.
  * Use `%{variable-name}` to indicate `neutron command` is a command template which would be executed multiple times according to variable‘s value.
* **\[variable-definition]**: Corresponding to `variable-name`, `variable-definition` tells the values used in the command template. The format of `variable-definition` is composed of `variable-name` and `values`. The `values` can be number range joint with `-` or string enumeration joint with `,`. For example:
  * `x:1-5`: [1 2 3 4 5]
  * `y:1-5,7,8,a,b,c`: [1 2 3 4 5 7 8 a b c]
  * `subnet:private-subnet,public-subnet`: [private-subnet public-subnet]

These 3 parts are divided with `--` and `++` as shown below.

### Help and Example

```
$ ./f5-oslbaasv2-batchops/dist/f5-oslbaasv2-batchops-darwin-amd64 --help
Usage:

    ./f5-oslbaasv2-batchops/dist/f5-oslbaasv2-batchops-darwin-amd64 [command arguments] -- <neutron command and arguments>[ ++ variable-definition]

Example:

    ./f5-oslbaasv2-batchops/dist/f5-oslbaasv2-batchops-darwin-amd64 --concurrency --output-filepath /dev/stdout \
    -- loadbalancer-create --name lb{x} {y} \
    ++ x:1-5 y:private-subnet,public-subnet

Command Arguments:

  -check-lb string
    	the loadbalancer name or id for checking execution status.
  -output-filepath string
    	output the result (default "/dev/stdout")
```

### Output and logs

Run command as:

```
$ ./f5-oslbaasv2-batchops/dist/f5-oslbaasv2-batchops-darwin-amd64 --output-filepath rlt.json -- loadbalancer-show lb-%{x} ++ x:1-2
```
logging:

```
2020/10/25 12:35:09 output to: rlt.json
2020/10/25 12:35:09 Command template: |loadbalancer-show lb-%{x}
2020/10/25 12:35:09 variables parsed as
2020/10/25 12:35:09          x: [1 2]
2020/10/25 12:35:09 neutron command: /Users/zong/PythonEnvs/openstack-client/bin/neutron
2020/10/25 12:35:09 Command(1/2): Start 'neutron lbaas-loadbalancer-show lb-1'
2020/10/25 12:35:12 Command(1/2): exits with: 0, executing time: 2996 ms
2020/10/25 12:35:13 Command(1/2): Checking Execution
2020/10/25 12:35:13 Command(1/2): Checked time: 2996 ms
2020/10/25 12:35:13 Command(2/2): Start 'neutron lbaas-loadbalancer-show lb-2'
2020/10/25 12:35:16 Command(2/2): exits with: 0, executing time: 2979 ms
2020/10/25 12:35:17 Command(2/2): Checking Execution
2020/10/25 12:35:17 Command(2/2): Checked time: 2979 ms
2020/10/25 12:35:17 Writen executions to file rlt.json: data-len:2208

---------------------- Execution Report ----------------------
1: neutron lbaas-loadbalancer-show lb-1 | Exited: 0 | Checked: lbaas-loadbalancer-show done | duration: 2996 ms
2: neutron lbaas-loadbalancer-show lb-2 | Exited: 0 | Checked: lbaas-loadbalancer-show done | duration: 2979 ms

Failed Command List:

-----------------------Execution Report End ---------------------
```

rlt.json

```
[
  {
    "seqnum": 1,
    "command": "neutron lbaas-loadbalancer-show lb-1",
    "output": {
      "admin_state_up": true,
      "description": "",
      "id": "7b7743eb-d70f-417b-83c6-9bb9b5f8e5df",
      "listeners": [
        {
          "id": "a6451fa5-db9a-4b2d-84b6-cb52a3cf6ea8"
        },
        {
          "id": "85a30551-ecc1-4c50-9f22-a5b6229fbf18"
        }
      ],
      "name": "lb-1",
      "operating_status": "ONLINE",
      "pools": [
        {
          "id": "87f60e6f-f1dc-4284-a5b8-2b265adeb0bc"
        }
      ],
      "provider": "f5networks",
      "provisioning_status": "ACTIVE",
      "tenant_id": "38ac07a46dad448cb93bec736ba89f1c",
      "vip_address": "10.0.0.25",
      "vip_port_id": "3c1ceab6-51f6-4da6-86be-e0dde1204f57",
      "vip_subnet_id": "b6fc5c77-d727-456e-bbd8-67a82534676c"
    },
    "error": "neutron CLI is deprecated and will be removed in the future. Use openstack CLI instead.\n",
    "exitcode": 0,
    "cmd_duration": 2996251436,
    "done_status": "lbaas-loadbalancer-show done",
    "done_duration": 2996257641,
    "done_loadbalancer": ""
  },
  {
    "seqnum": 2,
    "command": "neutron lbaas-loadbalancer-show lb-2",
    "output": {
      "admin_state_up": true,
      "description": "",
      "id": "2484e9ca-601d-4f02-9e11-0eb55ec36bf4",
      "listeners": [
        {
          "id": "614f93ca-3565-4bd5-ae6b-d24ff29021a0"
        },
        {
          "id": "416dc93f-bc5a-4b33-91d0-b12b8d09c726"
        }
      ],
      "name": "lb-2",
      "operating_status": "ONLINE",
      "pools": [
        {
          "id": "692490a6-ddfe-4834-9f07-8e84be44bd04"
        }
      ],
      "provider": "f5networks",
      "provisioning_status": "ACTIVE",
      "tenant_id": "38ac07a46dad448cb93bec736ba89f1c",
      "vip_address": "10.0.0.27",
      "vip_port_id": "b8bce411-2e50-4c75-a862-12e2942ee5ce",
      "vip_subnet_id": "b6fc5c77-d727-456e-bbd8-67a82534676c"
    },
    "error": "neutron CLI is deprecated and will be removed in the future. Use openstack CLI instead.\n",
    "exitcode": 0,
    "cmd_duration": 2979265336,
    "done_status": "lbaas-loadbalancer-show done",
    "done_duration": 2979267489,
    "done_loadbalancer": ""
  }
]
```