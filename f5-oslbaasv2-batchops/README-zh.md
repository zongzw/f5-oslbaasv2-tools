# f5-oslbaasv2-batchops

本项目旨在帮助使用者批量的执行`neutron`命令。通常在性能测试场景中会使用到。批量操作包括创建、更新、删除、查看 LBaaSv2 中的各种资源：loadbalancer/listener/pool/member/healthmonitor/l7policy/l7rule。

该命令的使用方式可以参见 [./scripts/create-lbaasv2-objects.sh](./scripts/create-lbaasv2-objects.sh) 封装。通常情况下可以修改脚本中的创建参数，执行脚本完成批量创建操作。

## 使用说明

此命令及参数包含三个部分：

* **\[command arguments]**: 命令及参数部分。此部分指明了命令本身的行为，比如

  `--output-filepath <filepath>` 指定了命令执行后的保存路径; 
  
  `--loadbalancer <lb id or name>` 指定了所所操作资源所属的loadbalancer。
* **\<neutron command>**: neutron 命令模板部分（可能包含变量声明）。此部分的语法和neutron原生命令（`neutron lbaas-*`）相同。但需要注意以下几点：
  * 为了方便使用，neutron命令的前缀部分`neutron ` 请不要包含在其中，例如： `neutron lbaas-loadbalancer-list`, 在使用中，只需要`lbaas-loadbalancer-list`。
  * 使用`%{variable-name}` 作为变量声明，这个声明会在实际执行过程中替换成具体的变量值，命令模板因此被替换成实际neutron命令执行多次。
* **\[variable-definition]**: 与上述 `variable-name`相对应，这是变量定义部分。`variable-definition`包含两部分：`variable-name` 和 `values`，格式和解析结果如下：
  * `x:1-5`: x 解析为 [1 2 3 4 5]
  * `y:1-5,7,8,a,b,c`: y 解析为 [1 2 3 4 5 7 8 a b c]
  * `subnet:private-subnet,public-subnet`: subnet 解析为 [private-subnet public-subnet]

这三部分以`--` 和 `++` 隔开，如下所示。

### 命令帮助及使用示例

```
$ ./f5-oslbaasv2-batchops/dist/f5-oslbaasv2-batchops-darwin-amd64 --help
Usage:

    ./f5-oslbaasv2-batchops/dist/f5-oslbaasv2-batchops-darwin-amd64 [command arguments] -- <neutron command and arguments>[ ++ variable-definition]

Example:

    ./f5-oslbaasv2-batchops/dist/f5-oslbaasv2-batchops-darwin-amd64 --output-filepath /dev/stdout \
    -- lbaas-loadbalancer-create --name lb%{x} %{y} \
    ++ x:1-5 y:private-subnet,public-subnet

Command Arguments:

  -loadbalancer string
    	the loadbalancer name or id for checking execution status.
  -output-filepath string
    	output the result (default "/dev/stdout")
```

### 结果输出及日志

按照如下方式执行此命令:

```
$ ./f5-oslbaasv2-batchops/dist/f5-oslbaasv2-batchops-darwin-amd64 --output-filepath rlt.json -- lbaas-loadbalancer-show lb-%{x} ++ x:1-2
```
响应日志为:

```
2020/10/25 12:35:09 output to: rlt.json
2020/10/25 12:35:09 Command template: |lbaas-loadbalancer-show lb-%{x}
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

最终结果输出至rlt.json：

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