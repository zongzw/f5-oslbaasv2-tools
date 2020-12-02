# f5-oslbaasv2-parselog

此命令可以用于分析F5Networks OpenStack LBaaSv2 的日志，包含Driver端和Agent端，从而通过日志得到所有LB对象对账单。
此处提到的Driver和Agent代码仓库如下：
* f5-openstack-lbaasv2-driver (https://github.com/F5Networks/f5-openstack-lbaasv2-driver) 
* f5-openstack-agent (https://github.com/F5Networks/F5-OPENSTACK-AGENT)

## 前置条件

使用此命令分析log之前需要确认日志级别DEBUG被启用:

* Neutron server 配置, 通常为 `/etc/neutron/neutron.conf`:

  ```
  #
  # From oslo.log
  #

  # If set to true, the logging level will be set to DEBUG instead of the default
  # INFO level. (boolean value)
  # Note: This option can be changed without restarting.
  debug = true
  ```

* Agent 配置, 通常为  `/etc/neutron/services/f5/f5-openstack-agent.ini`: 

  ```
  [DEFAULT]
  # Show debugging output in log (sets DEBUG log level output).
  debug = True
  ```

否则部分有用信息无法提取到。

但这不会造成此命令执行异常或失败。

多NeutronServer场景中，需要保持各个Server的时间同步（通常使用NTP实现），否则最终统计的时耗会不准确。

## 使用说明

详细的使用方法可以通过执行`f5-oslbaasv2-logtool --help`得到，如下所示。
```
# ./f5-oslbaasv2-logtool --help
Usage of ./f5-oslbaasv2-parselog/dist/f5-oslbaasv2-parselog-darwin-amd64:
  -logpath value
        The log path for analytics. It can be used multiple times. Path regex pattern can be used WITHIN "".
        e.g: --logpath /path/to/f5-openstack-agent.log --logpath "/var/log/neutron/f5-openstack-*.log" --logpath /var/log/neutron/server\*.log
  -max-line-length int
        Max line length in the parsing log. (default 524288)
  -output-filepath string
        Output the result to file, e.g: /path/to/result.csv (default "./result.csv")
  -test
        Program self test option..
```

其中，
* --logpath 支持正则表达式方式指明路径，需要注意的是，需要对\'\*'做转义：'\\*'(如上所示)，这样是为了避免路径被shell提前转义成具体的文件列表。
* --logpath指定的log文件必须为文本文件，不能是.gz 等压缩文件格式。
* `--output-filepath` 指定的输出文档格式.csv，包含以下数据列(后续会根据日志分析需求进一步扩展)：
  * req_id: 请求ID
  * object_id: 对象ID
  * object_type: 对象类型
  * operation_type: 操作类型
  * request_body: 请求内容
  * neutron_api_time: 请求调用到neutron api的时间点
  * call_f5driver_time: 请求调用到F5 Driver侧的时间点
  * rpc_f5agent_time: 请求调用到RPC的时间点
  * call_f5agent_time: 请求调用到F5 Agent的时间点
  * call_bigip_time: 请求调用到BIG-IP的时间点
  * update_status_time: 请求完成update_status时的时间点
  * dur_neu_drv: 从neutron 到 driver 的时间段
  * dur_drv_rpc: 从driver 到RPC的时间段
  * dur_rpc_agt: 从RPC 到agent 的时间段
  * dur_agt_upd: 从agent执行到完成的时间段
  * total: 总共执行的时间段（从neuron api 到update status）

## 日志格式参考

F5 Driver和Agent运行过程中产生的日志的格式定义可以在 `/etc/neutron/neutron.conf`中找到(搜`'logging'`):

```
# Format string to use for log messages with context. (string value)
logging_context_format_string = %(asctime)s.%(msecs)03d %(process)d %(levelname)s %(name)s [%(request_id)s %(user_identity)s] %(instance)s%(message)s

# Format string to use for log messages when context is undefined. (string
# value)
logging_default_format_string = %(asctime)s.%(msecs)03d %(process)d %(levelname)s %(name)s [-] %(instance)s%(message)s

# Additional data to append to log message when logging level for the message
# is DEBUG. (string value)
logging_debug_format_suffix = %(funcName)s %(pathname)s:%(lineno)d

# Prefix each line of exception output with this format. (string value)
logging_exception_prefix = %(asctime)s.%(msecs)03d %(process)d ERROR %(name)s %(instance)s

# Defines the format string for %(user_identity)s that is used in
# logging_context_format_string. (string value)
logging_user_identity_format = %(user)s %(tenant)s %(domain)s %(user_domain)s %(project_domain)s
```

## 命令限制

* 单行日志的长度最大为512K Bytes（目前已知最大长度为~64KB）。
* 此命令执行过程中会多线程读入各个`--logpath`文件内容（每个文件一个线程）。同样，分析过程也是多个线程并行执行以提高匹配效率。默认情况下不需要修改`nThrFilesMax` 和 `nThrParse`，这两个值已调至最优值。如修改，可以执行`./build.sh` 重新编译。
* 程序处理log 能力： > 50000 行/秒。