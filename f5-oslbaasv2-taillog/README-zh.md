# f5-oslbaasv2-taillog

此工具用于方便前端工程师在客户环境中收集特定时间段中对问题定位修复有帮助的日志。

## 使用说明

```
# ./f5-oslbaasv2-taillog-linux-amd64 --help
Usage of ./f5-oslbaasv2-taillog-linux-amd64:
  -begin-time string
        start datetime, format: 2006-01-02 15:04:05.000
  -end-time string
        end datetime, format: 2006-01-02 15:04:05.000
  -filter value
        filter keys, regexp supported.
  -logpath value
        log paths, regexp supported.
  -output-dirpath string
        output folder, will be created if not exists. (default ".")
```

此工具通过指明`--begin-time` 和 `--end-time`，确定要收集日志的时间段，也就是说程序运行后只关注此时间段内的日志，将内容拷贝到`--output-dirpath`指定的目录下，文件名同源文件名相同。

`--begin-time` 和 `--end-time`的格式为`2006-01-02 15:04:05.000`，OpenStack 和 F5 LBaaSv2 Agent的通用日志时间格式，通常这两个时间可以从日志文件中拷贝出来作为要收集的日志时间起止点。

如果指定了`--filter`参数，拷贝过程中会执行过滤操作，只有匹配到的日志条目才会写入到被收集文件中。`--filter` 参数可以被多次使用，多次出现的情况下，log条目只要匹配到任何一个`--filter`都会被收集。