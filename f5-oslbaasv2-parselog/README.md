# f5-oslbaasv2-parselog

This is a program(in golang) used to analyze logs from f5 OpenStack LBaaSv2 provider, includes:
* f5-openstack-lbaasv2-driver (https://github.com/F5Networks/f5-openstack-lbaasv2-driver) and 
* f5-openstack-agent (https://github.com/F5Networks/F5-OPENSTACK-AGENT)

## Requirement

You need to change the log level to debug:

* Neutron server configuration, generally `/etc/neutron/neutron.conf`:

  ```
  #
  # From oslo.log
  #

  # If set to true, the logging level will be set to DEBUG instead of the default
  # INFO level. (boolean value)
  # Note: This option can be changed without restarting.
  debug = true
  ```

* Agent configuration, generally `/etc/neutron/services/f5/f5-openstack-agent.ini`: 

  ```
  [DEFAULT]
  # Show debugging output in log (sets DEBUG log level output).
  debug = True
  ```

For multiple-neutron-server environment, please keep the datetime are same(NTP should be better), otherwise, the duration(s) may be not accurate.

## Usage

Run `./dist/f5-oslbaasv2-parselog-darwin-amd64 --help` for help. 
```
$ ./dist/f5-oslbaasv2-parselog-darwin-amd64 --help
Usage of ./dist/f5-oslbaasv2-parselog-darwin-amd64:
  -logpath value
    	The log path for analytics. It can be used multiple times. Path regex pattern can be used WITHIN "".
    	e.g: --logpath /path/to/f5-openstack-agent.log --logpath "/var/log/neutron/f5-openstack-*.log" --logpath /var/log/neutron/server\*.log
  -max-line-length int
    	Max line length in the parsing log. (default 524288)
  -output-elk string
    	Output[POST] the result to ELK system: http://1.2.3.4:20003
  -output-filepath string
    	Output the result to file, e.g: /path/to/result.csv (default "./result.csv")
  -test
    	Program self test option..
```

* --logpath supports regex file paths, however you should use '\\*'(shown above) to avoid path convertion by shell in front of execution of this program.
* The logs need to be plain-text, should not be .gz(geneated via logrotate tools for example).
* The result would be output to a file appointed by `--output-filepath` with .csv format. The title includes:
  * request_id,object_id,
  * agent_module,
  * request_body,object_type,operation_type,
  * user_id,tenant_id,
  * loadbalancer,result,
  * time series
    * time_neutron_api_controller_prepare_request_body,
    * time_neutron_lbaas_driver,
    * time_f5driver,
    * time_portcreated,
    * time_rpc,
    * time_f5agent,
    * time_update_status,
  * duration series
    * duration_neutron_total,
    * duration_portcreated,
    * duration_driver,
    * duration_rpc,
    * duration_agent,
    * duration_bigip,
    * duration_total,
  * bigip_request_count

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

* The single line length should be less than 512 * 1024 english characters. If the log is longer than that, use `--max-line-length` to enlarge it, usually double it to 1024 * 1024, if it's still not satisfied, enable it again via `--max-line-length`.
* The program will make dozens of threads for 1) reading from files(1 per log file appointed by `--logpath`) and 2) parsing log entries cocurrently.
* Logging handling performance: > 50000 lines/sec.
