# f5-oslbaasv2-batchops

本项目旨在帮助使用者批量的执行`neutron`命令。通常在性能测试场景中会使用到。批量操作包括创建、更新、删除、查看 LBaaSv2 中的各种资源：loadbalancer/listener/pool/member/healthmonitor/l7policy/l7rule

## 使用说明

此命令及参数包含三个部分：

* **\[command arguments]**: 命令及参数部分。此部分指明了命令本身的行为，比如`--output <filepath>`指定了命令执行后的保存路径。
* **\<neutron command>**: neutron 命令模板部分（可能包含变量声明）。此部分的语法和neutron原生命令（`neutron lbaas-*`）相同。但需要注意以下几点：
  * 为了方便使用，neutron命令的前缀部分`neutron lbaas-` 请不要包含在其中，例如： `neutron lbaas-loadbalancer-list`, 在使用中，只需要`loadbalancer-list`。
  * 使用`%{<variable-name>}` 作为变量声明，这个声明会在实际执行过程中替换成具体的变量值，命令模板因此被替换成实际neutron命令执行多次。
* **\[variable-definition]**: 与上述 `variable-name`相对应，这是变量定义部分。`variable-definition`包含两部分：`variable-name` 和 `values`，格式和解析结果如下：
  * `1-5`: [1 2 3 4 5]
  * `1-5,7,8,a,b,c`: [1 2 3 4 5 7 8 a b c]
  * `private-subnet,public-subnet`: [private-subnet public-subnet]

这三部分以`--` 和 `++` 隔开，如下所示。

### 命令帮助及使用示例

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

### 结果输出及日志

按照如下方式执行此命令:

```
# ./dist/f5-oslbaasv2-batchops-linux-amd64 --output rlt.json -- loadbalancer-show lb%{x} ++ x:1-5
```
响应日志为:

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

最终结果输出至rlt.json：

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