# f5-oslbaasv2-taillog

This is a program used to collect logs by given begin/end time and filters.

## Usage

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

By telling the program `--begin-time` and `--end-time`, the program will determine the time scope first and then copy the content of `--logpath` between the time scope to a new file  under `--output-dirpath` with the same file name.

The format of `--begin-time` and `--end-time` should be `2006-01-02 15:04:05.000` as shown above. Copy and paste them from log files appointed by `--logpath`.

What's more, if `--filter` is provided, the copying process will only happen when the log line match the `--filter` regexp. The `--filter` parameter can be used multiple times. In that case, copying will happen when anyone of  `--filter` matches.
