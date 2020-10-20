# f5-oslbaasv2-batchops

This is a program(in golang) to help running `neutron` commands in batch.
It is usually used to create dozens of loadbalancer resources, such as loadbalancer/listener/pool/member/healthmonitor/l7policy/l7rule for performance test.

## Usage

The command arguments includes 3 parts, as shown below help and examples:

* **\[command arguments]**: The arguments to control the command's behavior, like `--output <filepath>` tells the command where to save the result.
* **\<neutron command>**: The neutron command template(with variables declared inside if any). Its syntex complies with native neutron commands(`neutron lbaas-*`) but 
  * For ease of use, the prefix `neutron lbaas-` should NOT be included here. For example, `neutron lbaas-loadbalancer-list`, change to `loadbalancer-list` instead here.
  * Use `%{<variable-name>}` to indicate `neutron command` is a command template which would be executed multiple times according to variable‘s value.
* **\[variable-definition]**: Corresponding to `variable-name`, `variable-definition` tells the values used in the command template. The format of `variable-definition` is composed of `variable-name` and `values`. The `values` can be number range joint with `-` or string enumeration joint with `,`. For example:
  * `1-5`: [1 2 3 4 5]
  * `1-5,7,8,a,b,c`: [1 2 3 4 5 7 8 a b c]
  * `private-subnet,public-subnet`: [private-subnet public-subnet]

The 3 parts are divided with `--` and `++` as shown below.

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

