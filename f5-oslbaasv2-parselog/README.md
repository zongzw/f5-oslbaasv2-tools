# f5-oslbaasv2-logtool

This is a program(in golang) used to analyze logs from f5 OpenStack LBaaSv2 provider, includes:
* f5-openstack-lbaasv2-driver (https://github.com/F5Networks/f5-openstack-lbaasv2-driver) and 
* f5-openstack-agent (https://github.com/F5Networks/F5-OPENSTACK-AGENT)

## Requirement

You need to change the log level to debug `([DEFAULT] debug = True`) in agent configuration, generally `/etc/neutron/services/f5/f5-openstack-agent.ini`.

## Usage

Run `f5-oslbaasv2-logtool --help` for help. 
```
# ./f5-oslbaasv2-logtool --help
Usage of ./f5-oslbaasv2-logtool:
  -logpath value
    	The log path for analytics. It can be used multiple times. Path regex pattern can be used WITHIN "".
    	e.g: --logpath /path/to/f5-openstack-agent.log --logpath "/var/log/neutron/f5-openstack-*.log" --logpath /var/log/neutron/server\*.log
  -output-filepath string
    	Output the result to file, e.g: /path/to/result.csv
  -test
    	Program self test option..
```

* --logpath supports regex file paths, however you should use '\\*'(shown above) to avoid path convertion by shell in front of execution of this program.
* The logs need to be plain-text, should not be .gz(geneated via logrotate tools for example).
* The result would be output to a file appointed by `--output-filepath` with .csv format. The title includes:
  * req_id
  * object_id
  * object_type
  * operation_type
  * request_body
  * neutron_api_time
  * call_f5driver_time
  * rpc_f5agent_time
  * call_f5agent_time
  * call_bigip_time
  * update_status_time
  * dur_neu_drv
  * dur_drv_rpc
  * dur_rpc_agt
  * dur_agt_upd
  * total

## Logging Format Reference

The logging formats used by f5 driver and f5 agent are defined in `/etc/neutron/neutron.conf`(search `'logging'` in that file):

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

## Limitation

* The single line length should be less than 512 * 1024 english characters.
* The program will make dozens of threads for 1) reading from files(1 per log file appointed by `--logpath`) and 2) parsing log entries cocurrently, change it if necessary(`nThrFilesMax` and `nThrParse` variable in the code), then re-build it.
* Logging handling performance: 4000 lines/sec.