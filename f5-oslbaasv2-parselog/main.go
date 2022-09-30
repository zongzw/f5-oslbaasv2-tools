package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"syscall"
	"time"

	// "encoding/json"
	"path/filepath"
	"sync"

	// "regexp"
	"github.com/trivago/grok"
	// "github.com/google/uuid"
)

type arrayFlags []string

const fkTimeLayout = "2006-01-02 15:04:05"

// MatchHandler a structure storing pattern and corresponding handler.
type MatchHandler struct {
	KeyString string
	Pattern   string
	Function  func(values map[string]string) error
}

// RequestContext request context parsed from log
type RequestContext struct {
	RequestID string `json:"request_id"`

	// key information parsed from log
	ObjectID      string `json:"object_id"`
	AgentModule   string `json:"agent_module"`
	RequestBody   string `json:"request_body"`
	ObjectType    string `json:"object_type"`
	OperationType string `json:"operation_type"`
	UserID        string `json:"user_id"`
	TenantID      string `json:"tenant_id"`
	LoadBalancer  string `json:"loadbalancer"`
	Result        string `json:"result"`

	// phrase timestamp
	TimeNeutronAPIControllerTakeAction         string `json:"time_neutron_api_controller_take_action" my:"noprint"` // not native
	TimeNeutronAPIControllerPrepareRequestBody string `json:"time_neutron_api_controller_prepare_request_body"`     // native
	TimeNeutronAPIControllerDoCreate           string `json:"time_neutron_api_controller_do_create" my:"noprint"`   // not native
	TimeNeutronLBaaSPlugin                     string `json:"time_neutron_lbaas_plugin" my:"noprint"`               // not native
	TimeNeutronLBaaSDriver                     string `json:"time_neutron_lbaas_driver"`                            // not used; native
	TimeF5Driver                               string `json:"time_f5driver"`
	TimePortCreated                            string `json:"time_portcreated"`
	TimeRPC                                    string `json:"time_rpc"`
	TimeF5Agent                                string `json:"time_f5agent"`
	TimeUpdateStatus                           string `json:"time_update_status"`
	TimestampELK                               string `json:"@timestamp" my:"noprint"`

	// access dbv2  analytics metrics
	TmpDBBeginTime string `json:"time_db_begin" my:"noprint"`
	TmpDBEndTime   string `json:"time_db_end" my:"noprint"`

	// access bigip analytics metrics
	TmpBigipTime    string   `json:"bigip_request_time" my:"noprint"`
	TmpResponseTime string   `json:"bigip_response_time" my:"noprint"`
	TmpBigipMethod  string   `json:"bigip_request_method" my:"noprint"`
	TmpResponseCode string   `json:"bigip_response_code" my:"noprint"`
	BigipAccesses   []string `json:"bigip_accesses" my:"noprint"`

	// Calculated data
	DurationNeutronAction   time.Duration  `json:"duration_neutron_take_action" my:"noprint"`
	DurationNeutronPRB      time.Duration  `json:"duration_neutron_prepare_request_body" my:"noprint"`
	DurationNeutronDoCreate time.Duration  `json:"duration_neutron_do_create" my:"noprint"`
	DurationNeutronTotal    time.Duration  `json:"duration_neutron_total"`
	DurationPortCreated     time.Duration  `json:"duration_portcreated"`
	DurationF5Driver        time.Duration  `json:"duration_driver"`
	DurationRPC             time.Duration  `json:"duration_rpc"`
	DurationF5Agent         time.Duration  `json:"duration_agent"`
	DurationBigip           time.Duration  `json:"duration_bigip"`
	DurationTotal           time.Duration  `json:"duration_total"`
	BigipRequestCount       map[string]int `json:"bigip_request_count"`
}

var (
	logger         = log.New(os.Stdout, "", log.LstdFlags)
	outputFilePath = "./result.csv"
	outputELK      = ""

	pBasicFields = map[string]string{
		"UUID":      `[a-z0-9]{8}-([a-z0-9]{4}-){3}[a-z0-9]{12}`, // 6245c77d-5017-4657-b35b-7ab1d247112b
		"REQID":     `req-%{UUID}`,                               // req-8cadad28-8315-45ca-818c-6a229dfb73e1
		"DATETIME":  `\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}.\d{3}`, // 2020-09-27 19:22:54.486
		"MD5":       `[0-9a-z]{32}`,                              // 62c38230485b4794a8eedece5dac9192
		"JSON":      `\{.*\}`,                                    // {u'bandwidth_limit_rule': {u'max_kbps': 102400, u'direction': u'egress', u'max_burst_kbps': 102400}}
		"LBTYPE":    `(LoadBalancer|Listener|Pool|Member|HealthMonitor|L7Policy|L7Rule|ACLGroup)`,
		"LBTYPESTR": `(loadbalancer|listener|pool|member|health_monitor|l7policy|l7rule|acl_group)`,
		"ACTION":    `(create|update|delete)`,
		"WORD":      `\w+`, // [0-9a-zA-Z_] strings
		"NUM":       `\d+`, // 202 400 200
		"RESULT":    `(ACTIVE|ERROR)`,
	}

	pLBaaSv2 = map[string]MatchHandler{

		// 2021-02-17 23:43:34.177 10483 DEBUG neutron.api.v2.base [req-953ac2ec-57b7-42bd-8daa-ddd30035ef80 57efef668e094651bc2058f2c7b6a09c e04af77e23be443989be14e22240ea75 - default default] neutron.api.v2.base.Controller method create called with arguments () {'body': {u'loadbalancer': {u'vip_subnet_id': u'8d957135-5fd1-45cb-8884-a5ae0f12f2f4', u'bandwidth': 0, u'admin_state_up': True}}, 'request': <Request at 0x7fa250b4bf50 POST http://10.145.73.123:9696/v2.0/lbaas/loadbalancers>} wrapper /usr/lib/python2.7/site-packages/oslo_log/helpers.py:66

		"neutron_api_v2_base_controller_take_action": {
			KeyString: "neutron.api.v2.base",
			Pattern:   `%{DATETIME:time_neutron_api_controller_take_action} .* neutron.api.v2.base \[%{REQID:request_id} %{WORD:user_id} %{MD5:tenant_id} .*\] neutron.api.v2.base.Controller method %{ACTION:operation_type} called with arguments.*$`,
			Function:  nil,
		},

		// 2020-09-27 19:22:54.485 68316 DEBUG neutron.api.v2.base
		// [req-8cadad28-8315-45ca-818c-6a229dfb73e1 009ac6496334436a8eba8daa510ef659 62c38230485b4794a8eedece5dac9192 - default default] Request body:
		// {u'bandwidth_limit_rule': {u'max_kbps': 102400, u'direction': u'egress', u'max_burst_kbps': 102400}}
		// prepare_request_body /usr/lib/python2.7/site-packages/neutron/api/v2/base.py:713

		// %(user)s %(tenant)s %(domain)s %(user_domain)s %(project_domain)s
		"neutron_api_v2_base_controller_prepare_request_body": {
			KeyString: "neutron.api.v2.base",
			Pattern:   `%{DATETIME:time_neutron_api_controller_prepare_request_body} .* neutron.api.v2.base \[%{REQID:request_id} %{WORD:user_id} %{MD5:tenant_id} .*\] Request body: %{JSON:request_body} prepare_request_body .*$`,
			Function:  nil,
		},

		// 2021-02-17 23:43:34.203 10483 DEBUG neutron.api.v2.base [req-953ac2ec-57b7-42bd-8daa-ddd30035ef80 57efef668e094651bc2058f2c7b6a09c e04af77e23be443989be14e22240ea75 - default default] static method do_create called with arguments ({u'loadbalancer': {'description': '', u'admin_state_up': True, 'tenant_id': u'e04af77e23be443989be14e22240ea75', 'vip_address': <neutron_lib.constants.Sentinel object at 0x7fa257f16a10>, 'vip_network_id': <neutron_lib.constants.Sentinel object at 0x7fa257f16a10>, u'bandwidth': 0, 'flavor_id': <neutron_lib.constants.Sentinel object at 0x7fa257f16a10>, 'provider': <neutron_lib.constants.Sentinel object at 0x7fa257f16a10>, u'vip_subnet_id': u'8d957135-5fd1-45cb-8884-a5ae0f12f2f4', 'project_id': u'e04af77e23be443989be14e22240ea75', 'name': ''}},) {} wrapper /usr/lib/python2.7/site-packages/oslo_log/helpers.py:66
		"neutron_api_v2_base_controller_do_create": {
			KeyString: "neutron.api.v2.base",
			Pattern:   `%{DATETIME:time_neutron_api_controller_do_create} .* neutron.api.v2.base \[%{REQID:request_id} %{WORD:user_id} %{MD5:tenant_id} .*\] static method do_create called with arguments .*$`,
			Function:  nil,
		},

		// 2021-02-17 23:43:34.203 10483 DEBUG neutron_lbaas.services.loadbalancer.plugin [req-953ac2ec-57b7-42bd-8daa-ddd30035ef80 57efef668e094651bc2058f2c7b6a09c e04af77e23be443989be14e22240ea75 - default default] neutron_lbaas.services.loadbalancer.plugin.LoadBalancerPluginv2 method create_loadbalancer called with arguments (<neutron_lib.context.Context object at 0x7fa250ceae50>,) {'loadbalancer': {u'loadbalancer': {'description': '', u'admin_state_up': True, 'tenant_id': u'e04af77e23be443989be14e22240ea75', 'vip_address': <neutron_lib.constants.Sentinel object at 0x7fa257f16a10>, 'vip_network_id': <neutron_lib.constants.Sentinel object at 0x7fa257f16a10>, u'bandwidth': 0, 'flavor_id': <neutron_lib.constants.Sentinel object at 0x7fa257f16a10>, 'provider': <neutron_lib.constants.Sentinel object at 0x7fa257f16a10>, u'vip_subnet_id': u'8d957135-5fd1-45cb-8884-a5ae0f12f2f4', 'project_id': u'e04af77e23be443989be14e22240ea75', 'name': ''}}} wrapper /usr/lib/python2.7/site-packages/oslo_log/helpers.py:66
		"neutron_lbaas_plugin": {
			KeyString: "neutron_lbaas.services.loadbalancer.plugin",
			Pattern:   `%{DATETIME:time_neutron_lbaas_plugin} .* neutron_lbaas.services.loadbalancer.plugin \[%{REQID:request_id} %{WORD:user_id} %{MD5:tenant_id} .*\] neutron_lbaas.services.loadbalancer.plugin.LoadBalancerPluginv2 method %{WORD} called with arguments .*$`,
			Function:  nil,
		},

		// 2020-11-06 21:11:05.844 708196 INFO neutron_lbaas.services.loadbalancer.plugin
		// [req-b5b8896b-cfa2-4adc-b5c4-ebd986e24a5f a975df1b007d413c8ebc2e90d46232cf 94f2338bf383405db151c4784c0e358c - default default]
		// Calling driver operation ListenerManager.delete
		"neutron_lbaas_driver": {
			KeyString: "neutron_lbaas.services.loadbalancer.plugin",
			Pattern: `%{DATETIME:time_neutron_lbaas_driver} .* neutron_lbaas.services.loadbalancer.plugin \[%{REQID:request_id} .*\] ` +
				`Calling driver operation %{LBTYPE:object_type}Manager.%{ACTION:operation_type}.*$`,
			Function: nil,
		},

		// 05neu-core/server.log-1005:2020-10-05 10:20:17.251 117825 DEBUG f5lbaasdriver.v2.bigip.driver_v2
		// [req-92db71fb-8513-431b-ac79-5423a749b6d7 009ac6496334436a8eba8daa510ef659 62c38230485b4794a8eedece5dac9192 - default default]
		// f5lbaasdriver.v2.bigip.driver_v2.LoadBalancerManager method create called with arguments (<neutron_lib.context.Context object at 0x284cb250>,
		// <neutron_lbaas.services.loadbalancer.data_models.LoadBalancer object at 0xdb44250>) {}
		// wrapper /usr/lib/python2.7/site-packages/oslo_log/helpers.py:66
		"call_f5driver": {
			KeyString: "f5lbaasdriver.v2.bigip.driver_v2",
			Pattern: `%{DATETIME:time_f5driver} .* f5lbaasdriver.v2.bigip.driver_v2 \[%{REQID:request_id} .*\] ` +
				`f5lbaasdriver.v2.bigip.driver_v2.%{LBTYPE:object_type}Manager method %{ACTION:operation_type} called with .*$`,
			Function: nil,
		},

		// 2020-10-28 16:17:00.234 150581 DEBUG f5lbaasdriver.v2.bigip.driver_v2
		// [req-3b85ab54-c3c6-4032-9ff7-6a56233d27d7 a975df1b007d413c8ebc2e90d46232cf 94f2338bf383405db151c4784c0e358c - default default]
		// the port created here is: {'status': u'DOWN', 'binding:host_id': u'POD1_CORE', 'description': None, 'allowed_address_pairs': [], 'tags': [],
		// 'extra_dhcp_opts': [], 'updated_at': '2020-10-28T08:16:59Z', 'device_owner': u'network:f5lbaasv2', 'revision_number': 3, 'port_security_enabled':
		// False, 'binding:profile': {}, 'fixed_ips': [{'subnet_id': u'550ebc09-2836-4ead-adef-225dd849c426', 'ip_address': u'10.250.23.18'}],
		// 'id': u'ffb79c1e-db31-44e2-bd82-1eeb51b51b43', 'security_groups': [], 'device_id': u'1ce8e148-6e6e-4d84-b1ce-dc05c268ef9b', 'name':
		// u'fake_pool_port_1ce8e148-6e6e-4d84-b1ce-dc05c268ef9b', 'admin_state_up': True, 'network_id': u'0f35109d-8620-4e46-882f-63f4b2e87163',
		// 'tenant_id': u'94f2338bf383405db151c4784c0e358c', 'binding:vif_details': {}, 'binding:vnic_type': u'baremetal', 'binding:vif_type': 'binding_failed',
		// 'qos_policy_id': None, 'mac_address': u'fa:16:3e:25:8e:1e', 'project_id': u'94f2338bf383405db151c4784c0e358c', 'created_at': '2020-10-28T08:16:59Z'}
		// create /usr/lib/python2.7/site-packages/f5lbaasdriver/v2/bigip/driver_v2.py:727
		"create_port": {
			KeyString: "f5lbaasdriver.v2.bigip.driver_v2",
			Pattern: `%{DATETIME:time_portcreated} .* f5lbaasdriver.v2.bigip.driver_v2 \[%{REQID:request_id} .*\] ` +
				`the port created here is: .*$`,
			Function: nil,
		},

		// 2020-10-05 10:20:21.924 117825 DEBUG f5lbaasdriver.v2.bigip.agent_scheduler
		// [req-92db71fb-8513-431b-ac79-5423a749b6d7 009ac6496334436a8eba8daa510ef659 62c38230485b4794a8eedece5dac9192 - default default]
		// Loadbalancer e2d277f7-eca2-46a4-bf2c-655856fd8733 is scheduled to lbaas agent dc55e196-319a-4c82-b262-344f45415131 schedule
		// /usr/lib/python2.7/site-packages/f5lbaasdriver/v2/bigip/agent_scheduler.py:306
		// "agent_scheduled":

		// 2020-10-05 10:20:27.176 117825 DEBUG f5lbaasdriver.v2.bigip.agent_rpc [req-92db71fb-8
		// 513-431b-ac79-5423a749b6d7 009ac6496334436a8eba8daa510ef659 62c38230485b4794a8eedece5dac9192 - default default]
		// f5lbaasdriver.v2.bigip.agent_rpc.LBaaSv2AgentRPC method create_loadbalancer called with arguments (<neutron_lib.
		// context.Context object at 0x284cb250>, {'availability_zone_hints': [], 'description': '', 'admin_state_up': True
		// , 'tenant_id': '62c38230485b4794a8eedece5dac9192', 'provisioning_status': 'PENDING_CREATE', 'listeners': [], 'vi
		// p_subnet_id': 'd79ef712-c1e3-4860-9343-d1702b9976aa', 'vip_address': '10.230.44.15', 'vip_port_id': '5bcbe2d7-99
		// 4f-40de-87ab-07aa632f0133', 'provider': None, 'pools': [], 'id': 'e2d277f7-eca2-46a4-bf2c-655856fd8733', 'operat
		// ing_status': 'OFFLINE', 'name': 'JL-B01-POD1-CORE-LB-7'}, {'subnets': ...
		// : 'd79ef712-c1e3-4860-9343-d1702b9976aa', 'vip_address': '10.230.44.15', 'vip_port_id': '5bcbe2d7-994f-40de-87ab
		// -07aa632f0133', 'provider': None, 'pools': [], 'id': 'e2d277f7-eca2-46a4-bf2c-655856fd8733', 'operating_status':
		// 'OFFLINE', 'name': 'JL-B01-POD1-CORE-LB-7'}}, u'POD1_CORE3') {} wrapper /usr/lib/python2.7/site-packages/oslo_l
		// og/helpers.py:66
		"rpc_f5agent": {
			KeyString: "f5lbaasdriver.v2.bigip.agent_rpc",
			Pattern: `%{DATETIME:time_rpc} .* f5lbaasdriver.v2.bigip.agent_rpc \[%{REQID:request_id} .*\] ` +
				`f5lbaasdriver.v2.bigip.agent_rpc.LBaaSv2AgentRPC method %{ACTION}_%{LBTYPESTR} called with arguments ` +
				`.*? 'id': '%{UUID:object_id}'.*`,
			Function: nil,
		},

		// 2020-10-05 10:19:16.315 295263 DEBUG f5_openstack_agent.lbaasv2.drivers.bigip.agent_manager
		// [req-92db71fb-8513-431b-ac79-5423a749b6d7 009ac6496334436a8eba8daa510ef659 62c38230485b4794a8eedece5dac9192 - - -]
		// f5_openstack_agent.lbaasv2.drivers.bigip.agent_manager.LbaasAgentManager method create_loadbalancer called with arguments
		// ...
		// 7'}} wrapper /usr/lib/python2.7/site-packages/oslo_log/helpers.py:66
		"call_f5agent": {
			KeyString: "f5_openstack_agent.lbaasv2.drivers.bigip",
			Pattern: `%{DATETIME:time_f5agent} .* f5_openstack_agent.lbaasv2.drivers.bigip.%{WORD:agent_module} \[%{REQID:request_id} .*\] ` +
				`f5_openstack_agent.lbaasv2.drivers.bigip.%{WORD}.LbaasAgentManager method %{ACTION}_%{LBTYPESTR} ` +
				`called with arguments .*`,
			Function: nil,
		},

		// 2020-10-05 10:19:16.317 295263 DEBUG root [req-92db71fb-8513-431b-ac79-5423a749b6d7 009ac6496334436a8eba8daa510ef659
		// 62c38230485b4794a8eedece5dac9192 - - -] get WITH uri: https://10.216.177.8:443/mgmt/tm/sys/folder/~CORE_62c38230485b4794a8eedece5dac9192 AND
		// suffix:  AND kwargs: {} wrapper /usr/lib/python2.7/site-packages/icontrol/session.py:257
		"rest_call_bigip": {
			KeyString: "WITH uri: ",
			Pattern:   `%{DATETIME:bigip_request_time} .* \[%{REQID:request_id} .*\] %{WORD:bigip_request_method} WITH uri: .*icontrol/session.py.*`,
			Function:  SetAccessBIP,
		},

		// 2020-10-28 16:17:55.280 151202 DEBUG root [req-3b85ab54-c3c6-4032-9ff7-6a56233d27d7 a975df1b007d413c8ebc2e90d46232cf
		// 94f2338bf383405db151c4784c0e358c - - -] RESPONSE::STATUS: 200 Content-Type: application/json; charset=UTF-8 Content-Encoding: None
		"rest_bigip_response": {
			KeyString: "RESPONSE::STATUS: ",
			Pattern:   `%{DATETIME:bigip_response_time} .* \[%{REQID:request_id} .*\] RESPONSE::STATUS: %{NUM:bigip_response_code} .*`,
			Function:  SetBIPResponse,
		},

		// 2020-10-05 10:19:18.411 295263 DEBUG f5_openstack_agent.lbaasv2.drivers.bigip.plugin_rpc
		// [req-92db71fb-8513-431b-ac79-5423a749b6d7 009ac6496334436a8eba8daa510ef659 62c38230485b4794a8eedece5dac9192 - - -]
		// f5_openstack_agent.lbaasv2.drivers.bigip.plugin_rpc.LBaaSv2PluginRPC method update_loadbalancer_status called with arguments
		// (u'e2d277f7-eca2-46a4-bf2c-655856fd8733', 'ACTIVE', 'ONLINE', u'JL-B01-POD1-CORE-LB-7') {} wrapper
		// /usr/lib/python2.7/site-packages/oslo_log/helpers.py:66
		"update_loadbalancer_status": {
			KeyString: "f5_openstack_agent.lbaasv2.drivers.bigip.plugin_rpc",
			Pattern: `%{DATETIME:time_update_status} .* f5_openstack_agent.lbaasv2.drivers.bigip.plugin_rpc \[%{REQID:request_id} .*\].* ` +
				`method update_loadbalancer_status called with arguments.*%{UUID:loadbalancer}.*%{RESULT:result}.*`,
			Function: nil,
		},

		// "test_basic_pattern":
		// 	`%{LBTYPE:object_type}`,
	}

	linesFIFOSize = 50
	linesFIFO     = make([]string, linesFIFOSize)

	// ResultMap The final result.
	// key: request id.
	// value: map of captured variable and values.
	ResultMap = map[string]*RequestContext{}

	rltLock = &sync.Mutex{}
	bufLock = &sync.Mutex{}

	readDone = false

	nThrParse   = 100
	nThrReadMax = 10

	chThrFiles = make(chan bool, nThrReadMax)

	wgParse = sync.WaitGroup{}
	wgRead  = sync.WaitGroup{}

	debugSize = 50000

	validLinePattern = `\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}.\d{3} .*req-[a-z0-9]{8}-([a-z0-9]{4}-){3}[a-z0-9]{12}`
	regLine          = regexp.MustCompile(validLinePattern)

	chSignal = make(chan os.Signal)

	totalLines    = 0
	maxLineLength = 512 * 1024
)

func main() {

	var logpaths arrayFlags
	// var output_ts string
	var t bool

	signal.Notify(chSignal, syscall.SIGINT, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGQUIT)
	go signalHandler()

	flag.Var(&logpaths, "logpath",
		"The log path for analytics. It can be used multiple times. "+
			"Path regex pattern can be used WITHIN \"\".\n"+
			"e.g: --logpath /path/to/f5-openstack-agent.log "+
			"--logpath \"/var/log/neutron/f5-openstack-*.log\" "+
			"--logpath /var/log/neutron/server\\*.log")
	flag.StringVar(&outputFilePath, "output-filepath", outputFilePath,
		"Output the result to file, e.g: /path/to/result.csv")
	flag.StringVar(&outputELK, "output-elk", "", "Output[POST] the result to ELK system: http://1.2.3.4:20003")
	// TODO: output result to f5 telemetry analytics.
	// flag.StringVar(&output_ts, "output-ts", "./result.json",
	// 	"Output the result to f5-telemetry-analytics. e.g: http://1.1.1.1:200002")
	flag.BoolVar(&t, "test", false, "Program self test option..")
	flag.IntVar(&maxLineLength, "max-line-length", maxLineLength, "Max line length in the parsing log.")
	flag.Parse()

	if t {
		TestProg()
		return
	}

	fileHandlers, err := HandleArguments(logpaths, outputFilePath)
	if err != nil {
		logger.Fatal(err)
	}
	for _, f := range fileHandlers {
		defer f.Close()
	}

	g, e := MakeGrok()
	if e != nil {
		logger.Fatal(e)
	}

	totalBegin := time.Now()
	for i := 0; i < nThrParse; i++ {
		wgParse.Add(1)
		go Parse(g)
	}

	for _, f := range fileHandlers {
		wgRead.Add(1)
		go Read(f)
	}

	wgRead.Wait()
	readDone = true
	wgParse.Wait()

	totalEnd := time.Now()

	logger.Println()
	logger.Printf("Total read %d lines, cost time: %f seconds", totalLines, totalEnd.Sub(totalBegin).Seconds())
	logger.Println()

	CalculateDuration()

	OutputResult(outputFilePath)
}

func signalHandler() {
	<-chSignal

	CalculateDuration()
	OutputResult(outputFilePath)
	os.Exit(1)
}

// Parse parse lines
func Parse(g *grok.Grok) {
	defer wgParse.Done()

	for {
		if len(linesFIFO) == 0 {
			if readDone {
				break
			}
			time.Sleep(time.Duration(5) * time.Millisecond)
			continue
		}

		t := ""
		bufLock.Lock()
		if len(linesFIFO) != 0 {
			t = linesFIFO[0]
			linesFIFO = linesFIFO[1:]
		}
		bufLock.Unlock()

		if t != "" {
			for k := range pLBaaSv2 {
				if Parse2Result(g, k, t) {
					break
				}
			}
		}
	}
}

// Parse2Result parse text with k pattern.
func Parse2Result(g *grok.Grok, k string, text string) bool {
	if !strings.Contains(text, pLBaaSv2[k].KeyString) {
		return false
	}

	values, err := g.ParseString(fmt.Sprintf("%%{%s}", k), text)
	if err != nil {
		logger.Fatal(err)
	}
	if len(values) == 0 {
		return false
	}

	if _, ok := values["request_id"]; !ok {
		logger.Fatalf("Abnormal thing happens. No request_id matched in log")
	}

	rltLock.Lock()
	defer rltLock.Unlock()

	if err = DefaultSet(values); err != nil {
		logger.Printf("Error setting value %s: %s\n", values, err.Error())
	}

	setFunc := pLBaaSv2[k].Function
	if setFunc != nil {
		if err = setFunc(values); err != nil {
			logger.Printf("Error customizing setting %s: %s\n", values, err.Error())
		}
	}

	return true
}

// SetAccessBIP additional setting to accessing bigip
func SetAccessBIP(values map[string]string) error {
	reqID := values["request_id"]

	rc := ResultMap[reqID]
	m := rc.TmpBigipMethod
	rc.BigipRequestCount[m]++
	rc.BigipAccesses = append(rc.BigipAccesses,
		fmt.Sprintf("%s|%s|%s", rc.TmpBigipTime, "request", rc.TmpBigipMethod))

	return nil
}

// SetBIPResponse additional setting for bigip response
func SetBIPResponse(values map[string]string) error {
	reqID := values["request_id"]

	rc := ResultMap[reqID]

	// Use 'reply' not 'response' here:
	// 	If the 'reply' time is as same as next 'request' time,
	// 	the 'reply' entry will be sorted before next 'request' one, like
	// ...
	// [6] "2020-11-05 03:07:22.416|request|get"
	// [7] "2020-11-05 03:07:22.482|reply|200"			<- this 'reply'
	// [8] "2020-11-05 03:07:22.482|request|patch"		<- next 'request'
	// [9] "2020-11-05 03:07:22.584|reply|200"
	// ...
	rc.BigipAccesses = append(rc.BigipAccesses,
		fmt.Sprintf("%s|%s|%s", rc.TmpResponseTime, "reply", rc.TmpResponseCode))

	return nil
}

// DefaultSet set values into ResultMap
func DefaultSet(values map[string]string) error {

	reqID := values["request_id"]
	if _, ok := ResultMap[reqID]; !ok {
		ResultMap[reqID] = &RequestContext{
			BigipRequestCount: map[string]int{},
		}
	}

	bValues, _ := json.Marshal(values)
	rc := ResultMap[reqID]
	if err := json.Unmarshal(bValues, rc); err != nil {
		return err
	}

	return nil
}

// Read lines from f to linesFIFO
func Read(f *os.File) {
	chThrFiles <- true
	defer func() {
		<-chThrFiles
		wgRead.Done()
	}()
	log.Printf("Start to reading %s\n", f.Name())

	scanner := bufio.NewScanner(f)
	maxCapacity := maxLineLength
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	fs := time.Now().UnixNano()
	lines := 0

	for {
		scanned := scanner.Scan()
		if !scanned {
			if err := scanner.Err(); err != nil {
				logger.Printf("Error happens at %s line %d.", f.Name(), lines+1)
				if strings.HasPrefix(err.Error(), "bufio.Scanner: token too long") {
					logger.Println()
					logger.Printf("Use --max-line-length to enlarge the buffer size to double size for another try: %d", maxCapacity*2)
					logger.Println()
				}
				logger.Fatal(err)

			} else {
				fe := time.Now().UnixNano()
				logger.Printf("Done of read from file %s, lines: %d, total time: %v ms \n",
					f.Name(), lines, (fe-fs)/1e6)
				break
			}
		}
		lines++
		if lines%debugSize == 0 {
			logger.Printf("%s, read lines %d, len of linesFIFO: %d\n", f.Name(), lines, len(linesFIFO))
		}
		if !regLine.MatchString(scanner.Text()) {
			continue
		}
		bufLock.Lock()
		linesFIFO = append(linesFIFO, scanner.Text())
		bufLock.Unlock()

		if len(linesFIFO) > linesFIFOSize {
			time.Sleep(time.Duration(50) * time.Microsecond)
		}
	}

	totalLines = totalLines + lines
}

// OutputResultToELK Output the result to ELK endpoint if set.
func OutputResultToELK() {
	if outputELK == "" {
		return
	}

	logger.Printf("Sending metrics to %s ...", outputELK)
	elk, err := url.Parse(outputELK)
	if err != nil {
		logger.Fatalf("invalid elk link: %s", err)
	}
	port := ""
	if !strings.Contains(elk.Host, ":") {
		port = ":20003"
	}

	timeout := 10 * time.Second
	_, err = net.DialTimeout("tcp", fmt.Sprintf("%s%s", elk.Host, port), timeout)
	if err != nil {
		logger.Fatalf("Site unreachable, error: %s", err)
	}

	count := 0
	for _, v := range ResultMap {
		count++
		jd, _ := json.Marshal(v)
		resp, err := http.Post(outputELK, "application/json", bytes.NewReader(jd))
		if err != nil || int(resp.StatusCode/200) != 1 {
			logger.Printf("Failed to post %s: %s", jd, err)
		}
	}

	logger.Printf("Metric count: %d", count)
}

// OutputResult output the ResultMap to file and ELK if set.
func OutputResult(filepath string) {

	OutputResultToELK()

	fw, e := os.Create(filepath)
	if e != nil {
		logger.Fatal(e)
	}
	defer fw.Close()

	titleLine := []string{}
	t := reflect.TypeOf(RequestContext{})
	for i := 0; i < t.NumField(); i++ {
		if !strings.Contains(t.Field(i).Tag.Get("my"), "noprint") {
			continue
		}
		titleLine = append(titleLine, t.Field(i).Tag.Get("json"))
	}

	_, e = fw.WriteString(strings.Join(titleLine, ",") + "\n")
	if e != nil {
		logger.Fatal(e)
	}

	for _, pRC := range ResultMap {
		dataLine := []string{}
		v := reflect.ValueOf(*pRC)
		t := reflect.TypeOf(RequestContext{})
		for i := 0; i < v.NumField(); i++ {
			if !strings.Contains(t.Field(i).Tag.Get("my"), "noprint") {
				continue
			}
			f := v.Field(i)
			switch f.Kind() {
			case reflect.String:
				dataItem := fmt.Sprintf("\"%s\"", f.String())
				dataItem = strings.ReplaceAll(dataItem, ",", " ")
				dataLine = append(dataLine, dataItem)
			case reflect.Int64: // support only time.Duration for now.
				dataLine = append(dataLine, fmt.Sprintf("%d", f.Int()/1e6))
			case reflect.Map: // support only map[string]int for now.
				ks := f.MapKeys()
				dl := ""
				for _, k := range ks {
					dl = dl + fmt.Sprintf("%s:%d ", k.String(), f.MapIndex(k).Int())
				}
				dataLine = append(dataLine, dl)
			default:
				dataLine = append(dataLine, fmt.Sprintf("%s", f.Interface()))
			}
		}

		line := strings.Join(dataLine, ",") + "\n"

		_, e := fw.WriteString(line)
		if e != nil {
			logger.Fatal(e)
		}
	}
}

// MakeGrok make a gork.Grok
func MakeGrok() (*grok.Grok, error) {
	pattern := map[string]string{}

	for k, v := range pBasicFields {
		pattern[k] = v
	}
	for k, v := range pLBaaSv2 {
		pattern[k] = v.Pattern
	}

	g, e := grok.New(grok.Config{
		NamedCapturesOnly: true,
		Patterns:          pattern,
	})
	return g, e
}

// HandleArguments handle arguments.
func HandleArguments(logpaths []string, outputFilePath string) ([]*os.File, error) {

	// handle output file for result.
	if outputFilePath == "" {
		return nil, fmt.Errorf("--output-filepath should be appointed")
	}
	dir, err := filepath.Abs(filepath.Dir(outputFilePath))
	if err != nil {
		return nil, err
	}

	_, err = os.Stat(dir)
	if os.IsNotExist(err) {
		return nil, err
	}

	_, err = os.Stat(outputFilePath)
	if err == nil {
		return nil, fmt.Errorf("file %s already exists", outputFilePath)
	}

	var fileHandlers []*os.File

	// parse regex paths
	paths := []string{}
	pathOK := true
	for _, n := range logpaths {
		p := string(n)
		ms, e := filepath.Glob(p)
		if e != nil {
			logger.Printf("Invalid file path or path pattern: %s\n", p)
			pathOK = false
		}
		paths = append(paths, ms...)
	}
	if !pathOK {
		return nil, fmt.Errorf("invalid file path detected, cannot to continue")
	}

	// find absolute paths
	pathOK = true
	for i, p := range paths {
		absP, err := filepath.Abs(p)
		if err != nil {
			logger.Printf("Cannot determine the absolute path for %s\n", p)
			pathOK = false
		}
		paths[i] = absP
	}
	if !pathOK {
		return nil, fmt.Errorf("invalid paths found while getting the absolute path. Cannot continue")
	}

	// remove duplicate paths
	noDupPaths := []string{}
	for _, p := range paths {
		if !IsContains(noDupPaths, p) {
			noDupPaths = append(noDupPaths, p)
		}
	}
	paths = noDupPaths
	logger.Printf("Handling files: %s\n", paths)

	// open as *os.File for reading
	pathOK = true
	for _, n := range paths {
		p := string(n)
		f, _ := os.Stat(string(p))

		if os.IsNotExist(err) {
			logger.Println(err.Error())
			pathOK = false
		}

		if f.IsDir() {
			logger.Printf("%s should be a file, not a directory, ignore it.\n", f.Name())
			continue
		}

		fr, err := os.Open(p)
		if err != nil {
			return nil, err
		}
		fileHandlers = append(fileHandlers, fr)
	}
	if !pathOK {
		return nil, fmt.Errorf("invalid path(s) provided")
	}

	return fileHandlers, nil
}

// FKTheTime parse the time string to time.Time
func FKTheTime(datm string) time.Time {
	dt, _ := time.Parse(fkTimeLayout, datm)
	return dt
}

// CalculateDuration calculate the duration.
func CalculateDuration() {
	for _, rc := range ResultMap {
		tNeutronAction := FKTheTime(rc.TimeNeutronAPIControllerTakeAction)
		tNeutronPRB := FKTheTime(rc.TimeNeutronAPIControllerPrepareRequestBody)
		tNeutronDoCreate := FKTheTime(rc.TimeNeutronAPIControllerDoCreate)
		tNeutronPlugin := FKTheTime(rc.TimeNeutronLBaaSPlugin)
		tDriver := FKTheTime(rc.TimeF5Driver)
		tPortCreated := FKTheTime(rc.TimePortCreated)
		tRPC := FKTheTime(rc.TimeRPC)
		tAgent := FKTheTime(rc.TimeF5Agent)
		tUpdate := FKTheTime(rc.TimeUpdateStatus)

		rc.TimestampELK = fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02d.%03dZ",
			tNeutronPRB.Year(), tNeutronPRB.Month(), tNeutronPRB.Day(),
			tNeutronPRB.Hour(), tNeutronPRB.Minute(), tNeutronPRB.Second(), tNeutronPRB.Nanosecond()/1e6)
		rc.DurationNeutronAction = tNeutronPRB.Sub(tNeutronAction)
		rc.DurationNeutronPRB = tNeutronDoCreate.Sub(tNeutronPRB)
		rc.DurationNeutronDoCreate = tNeutronPlugin.Sub(tNeutronDoCreate)
		rc.DurationNeutronTotal = tDriver.Sub(tNeutronPRB)
		rc.DurationF5Driver = tRPC.Sub(tDriver)
		rc.DurationPortCreated = tPortCreated.Sub(tDriver)
		rc.DurationRPC = tAgent.Sub(tRPC)
		rc.DurationF5Agent = tUpdate.Sub(tAgent)
		rc.DurationTotal = tUpdate.Sub(tNeutronPRB)

		biplen := len(rc.BigipAccesses)
		if biplen%2 != 0 {
			rc.DurationBigip = time.Duration(-2) * time.Millisecond
			continue
		}
		sort.Strings(rc.BigipAccesses)
		rc.DurationBigip = time.Duration(0) * time.Millisecond

		for i := 0; i < len(rc.BigipAccesses)-1; i = i + 2 {
			tms := strings.Split(rc.BigipAccesses[i], "|")
			tme := strings.Split(rc.BigipAccesses[i+1], "|")
			if (tms[1] != "request" || tme[1] != "reply") && tms[0] != tme[0] {
				rc.DurationBigip = time.Duration(-3) * time.Millisecond
				break
			}

			rc.DurationBigip = rc.DurationBigip + FKTheTime(tme[0]).Sub(FKTheTime(tms[0]))
		}
	}
}

func (i *arrayFlags) String() string {
	return fmt.Sprint(*i)
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

// IsContains check if a list contains the very item.
func IsContains(items []string, item string) bool {
	for _, n := range items {
		if n == item {
			return true
		}
	}
	return false
}

// TestProg test program
func TestProg() {
	g, e := MakeGrok()
	if e != nil {
		logger.Fatal(e)
	}

	for k, t := range TestCases() {
		for _, tn := range t {
			v, e := TestParse(k, tn, g)
			DebugTesting(v, e)
		}
	}
}

// DebugTesting debug testing
func DebugTesting(values map[string]string, e error) {
	if e != nil {
		logger.Println(e.Error())
		return
	}

	if len(values) == 0 {
		logger.Println("no match for this pattern.")
		return
	}

	for k, v := range values {
		logger.Printf("%+25s: %s\n", k, v)
	}
	logger.Println()
}

// TestParse test parse
func TestParse(k string, v string, g *grok.Grok) (map[string]string, error) {
	return g.ParseString(fmt.Sprintf("%%{%s}", k), v)
}

// TestCases test cases
func TestCases() map[string][]string {
	return map[string][]string{
		"neutron_api_v2_base_controller_take_action": {
			// loadbalancer
			`2021-02-17 23:43:34.177 10483 DEBUG neutron.api.v2.base [req-953ac2ec-57b7-42bd-8daa-ddd30035ef80 57efef668e094651bc2058f2c7b6a09c e04af77e23be443989be14e22240ea75 - default default] neutron.api.v2.base.Controller method create called with arguments () {'body': {u'loadbalancer': {u'vip_subnet_id': u'8d957135-5fd1-45cb-8884-a5ae0f12f2f4', u'bandwidth': 0, u'admin_state_up': True}}, 'request': <Request at 0x7fa250b4bf50 POST http://10.145.73.123:9696/v2.0/lbaas/loadbalancers>} wrapper /usr/lib/python2.7/site-packages/oslo_log/helpers.py:66`,
		},

		"neutron_api_v2_base_controller_prepare_request_body": {
			// loadbalancer
			`2020-10-05 10:20:15.791 117825 DEBUG neutron.api.v2.base [req-92db71fb-8513-431b-ac79-5423a749b6d7 009ac6496334436a8eba8daa510ef659 62c38230485b4794a8eedece5dac9192 - default default] Request body: {u'loadbalancer': {u'vip_subnet_id': u'd79ef712-c1e3-4860-9343-d1702b9976aa', u'provider': u'core', u'name': u'JL-B01-POD1-CORE-LB-7', u'admin_state_up': True}} prepare_request_body /usr/lib/python2.7/site-packages/neutron/api/v2/base.py:713`,
			// member
			`2020-10-05 14:50:24.795 117812 DEBUG neutron.api.v2.base [req-be08ea84-f721-46da-b24e-6e2c249af84e 009ac6496334436a8eba8daa510ef659 62c38230485b4794a8eedece5dac9192 - default default] Request body: {u'member': {u'subnet_id': u'5ee954be-8a76-4e42-b7a9-13a08e5330ce', u'address': u'10.230.3.39', u'protocol_port': 39130, u'weight': 5, u'admin_state_up': True}} prepare_request_body /usr/lib/python2.7/site-packages/neutron/api/v2/base.py:713`,
		},

		"neutron_api_v2_base_controller_do_create": {
			// loadbalancer
			`2021-02-17 23:43:34.203 10483 DEBUG neutron.api.v2.base [req-953ac2ec-57b7-42bd-8daa-ddd30035ef80 57efef668e094651bc2058f2c7b6a09c e04af77e23be443989be14e22240ea75 - default default] static method do_create called with arguments ({u'loadbalancer': {'description': '', u'admin_state_up': True, 'tenant_id': u'e04af77e23be443989be14e22240ea75', 'vip_address': <neutron_lib.constants.Sentinel object at 0x7fa257f16a10>, 'vip_network_id': <neutron_lib.constants.Sentinel object at 0x7fa257f16a10>, u'bandwidth': 0, 'flavor_id': <neutron_lib.constants.Sentinel object at 0x7fa257f16a10>, 'provider': <neutron_lib.constants.Sentinel object at 0x7fa257f16a10>, u'vip_subnet_id': u'8d957135-5fd1-45cb-8884-a5ae0f12f2f4', 'project_id': u'e04af77e23be443989be14e22240ea75', 'name': ''}},) {} wrapper /usr/lib/python2.7/site-packages/oslo_log/helpers.py:66`,
		},

		"neutron_lbaas_plugin": {
			// loadbalancer
			`2021-02-17 23:43:34.203 10483 DEBUG neutron_lbaas.services.loadbalancer.plugin [req-953ac2ec-57b7-42bd-8daa-ddd30035ef80 57efef668e094651bc2058f2c7b6a09c e04af77e23be443989be14e22240ea75 - default default] neutron_lbaas.services.loadbalancer.plugin.LoadBalancerPluginv2 method create_loadbalancer called with arguments (<neutron_lib.context.Context object at 0x7fa250ceae50>,) {'loadbalancer': {u'loadbalancer': {'description': '', u'admin_state_up': True, 'tenant_id': u'e04af77e23be443989be14e22240ea75', 'vip_address': <neutron_lib.constants.Sentinel object at 0x7fa257f16a10>, 'vip_network_id': <neutron_lib.constants.Sentinel object at 0x7fa257f16a10>, u'bandwidth': 0, 'flavor_id': <neutron_lib.constants.Sentinel object at 0x7fa257f16a10>, 'provider': <neutron_lib.constants.Sentinel object at 0x7fa257f16a10>, u'vip_subnet_id': u'8d957135-5fd1-45cb-8884-a5ae0f12f2f4', 'project_id': u'e04af77e23be443989be14e22240ea75', 'name': ''}}} wrapper /usr/lib/python2.7/site-packages/oslo_log/helpers.py:66`,
		},

		"neutron_lbaas_driver": {
			// member.create
			"2020-11-05 03:05:13.382 423178 INFO neutron_lbaas.services.loadbalancer.plugin [req-784572e6-4622-477e-8500-ab43539b86de a975df1b007d413c8ebc2e90d46232cf 0699110021c743249033aad76967f42f - default default] Calling driver operation MemberManager.create",
			// listener.delete
			"2020-11-06 21:11:05.844 708196 INFO neutron_lbaas.services.loadbalancer.plugin [req-b5b8896b-cfa2-4adc-b5c4-ebd986e24a5f a975df1b007d413c8ebc2e90d46232cf 94f2338bf383405db151c4784c0e358c - default default] Calling driver operation ListenerManager.delete",
		},

		"call_f5driver": {
			// loadbalancer
			`2020-10-05 10:20:17.251 117825 DEBUG f5lbaasdriver.v2.bigip.driver_v2 [req-92db71fb-8513-431b-ac79-5423a749b6d7 009ac6496334436a8eba8daa510ef659 62c38230485b4794a8eedece5dac9192 - default default] f5lbaasdriver.v2.bigip.driver_v2.LoadBalancerManager method create called with arguments (<neutron_lib.context.Context object at 0x284cb250>, <neutron_lbaas.services.loadbalancer.data_models.LoadBalancer object at 0xdb44250>) {} wrapper /usr/lib/python2.7/site-packages/oslo_log/helpers.py:66`,
			// member
			`2020-10-05 14:50:28.214 117812 DEBUG f5lbaasdriver.v2.bigip.driver_v2 [req-be08ea84-f721-46da-b24e-6e2c249af84e 009ac6496334436a8eba8daa510ef659 62c38230485b4794a8eedece5dac9192 - default default] f5lbaasdriver.v2.bigip.driver_v2.MemberManager method create called with arguments (<neutron_lib.context.Context object at 0x1310cc90>, <neutron_lbaas.services.loadbalancer.data_models.Member object at 0x286ed750>) {} wrapper /usr/lib/python2.7/site-packages/oslo_log/helpers.py:66`,
		},

		"create_port": {
			// member create port
			`2020-11-05 03:06:08.342 423178 DEBUG f5lbaasdriver.v2.bigip.driver_v2 [req-784572e6-4622-477e-8500-ab43539b86de a975df1b007d413c8ebc2e90d46232cf 0699110021c743249033aad76967f42f - default default] the port created here is: {'status': u'DOWN', 'binding:host_id': u'POD1_CORE', 'description': None, 'allowed_address_pairs': [], 'tags': [], 'extra_dhcp_opts': [], 'updated_at': '2020-11-04T19:06:07Z', 'device_owner': u'network:f5lbaasv2', 'revision_number': 3, 'port_security_enabled': False, 'binding:profile': {}, 'fixed_ips': [{'subnet_id': u'550ebc09-2836-4ead-adef-225dd849c426', 'ip_address': u'10.250.23.9'}], 'id': u'a3a775f4-a20f-450d-bfe9-514bc1592304', 'security_groups': [], 'device_id': u'502e2b95-247b-42b8-9bf5-bb1d5479cbd1', 'name': u'fake_pool_port_502e2b95-247b-42b8-9bf5-bb1d5479cbd1', 'admin_state_up': True, 'network_id': u'0f35109d-8620-4e46-882f-63f4b2e87163', 'tenant_id': u'94f2338bf383405db151c4784c0e358c', 'binding:vif_details': {}, 'binding:vnic_type': u'baremetal', 'binding:vif_type': 'binding_failed', 'qos_policy_id': None, 'mac_address': u'fa:16:3e:41:16:cb', 'project_id': u'94f2338bf383405db151c4784c0e358c', 'created_at': '2020-11-04T19:06:07Z'} create /usr/lib/python2.7/site-packages/f5lbaasdriver/v2/bigip/driver_v2.py:729`,
		},

		"rpc_f5agent": {
			// loadbalancer
			`2020-10-05 10:20:27.176 117825 DEBUG f5lbaasdriver.v2.bigip.agent_rpc [req-92db71fb-8513-431b-ac79-5423a749b6d7 009ac6496334436a8eba8daa510ef659 62c38230485b4794a8eedece5dac9192 - default default] f5lbaasdriver.v2.bigip.agent_rpc.LBaaSv2AgentRPC method create_loadbalancer called with arguments (<neutron_lib.context.Context object at 0x284cb250>, {'availability_zone_hints': [], 'description': '', 'admin_state_up': True, 'tenant_id': '62c38230485b4794a8eedece5dac9192', 'provisioning_status': 'PENDING_CREATE', 'listeners': [], 'vip_subnet_id': 'd79ef712-c1e3-4860-9343-d1702b9976aa', 'vip_address': '10.230.44.15', 'vip_port_id': '5bcbe2d7-994f-40de-87ab-07aa632f0133', 'provider': None, 'pools': [], 'id': 'e2d277f7-eca2-46a4-bf2c-655856fd8733', 'operating_status': 'OFFLINE', 'name': 'JL-B01-POD1-CORE-LB-7'}, {'subnets': {u'd79ef71...OD1_CORE3') {} wrapper /usr/lib/python2.7/site-packages/oslo_log/helpers.py:66`,

			// member
			`2020-10-05 14:51:54.445 117812 DEBUG f5lbaasdriver.v2.bigip.agent_rpc [req-be08ea84-f721-46da-b24e-6e2c249af84e 009ac6496334436a8eba8daa510ef659 62c38230485b4794a8eedece5dac9192 - default default] f5lbaasdriver.v2.bigip.agent_rpc.LBaaSv2AgentRPC method create_member called with arguments (<neutron_lib.context.Context object at 0x1310cc90>, {'name': '', 'weight': 5, 'provisioning_status': 'PENDING_CREATE', 'subnet_id': '5ee954be-8a76-4e42-b7a9-13a08e5330ce', 'tenant_id': '62c38230485b4794a8eedece5dac9192', 'admin_state_up': True, 'pool_id': '100858a1-8ba9-496c-9cb4-7d1143431ce8', 'address': '10.230.3.39', 'protocol_port': 39130, 'id': '551b7992-273f-4923-94f2-57b12a715c15', 'operating_status': 'OFFLINE'}, {'subne...18273-1f5e-4be2-a263-ce37823a7773', 'operating_status': 'ONLINE', 'name': 'JL-B01-POD1-CORE-LB-1'}}, u'POD1_CORE') {} wrapper /usr/lib/python2.7/site-packages/oslo_log/helpers.py:66`,
		},

		"call_f5agent": {
			// loadbalancer
			`2020-10-05 10:19:16.315 295263 DEBUG f5_openstack_agent.lbaasv2.drivers.bigip.agent_manager [req-92db71fb-8513-431b-ac79-5423a749b6d7 009ac6496334436a8eba8daa510ef659 62c38230485b4794a8eedece5dac9192 - - -] f5_openstack_agent.lbaasv2.drivers.bigip.agent_manager.LbaasAgentManager method create_loadbalancer called with arguments (<neutron_lib.context.Context object at 0x7351290>,) {u'service': {u'subnets': {u'd79ef712-c1e3-4860-9343-d1702b9976aa': {u'updated_at': u'2020-09-25T05:29:56Z', u'ipv6_ra_mode': None, u'allocation_pools': [{u'start': u'10.230.44.2', u'end': u'10.230.44.30'}], u'host_routes': [], u'revision_number': 1, u'ipv6_address_mode': None, u'id': u'd79ef712-c1e3-4860-9343-d1702b9976aa', u'available_ips': [{u'start': u'10.230.44.3', u'end': u'10.230.44.3'}, {u'start': u'10.230.44.10', u'end': u'10.230.44.12'}, {u'start': u'10.230.44.14', u'end': u'10.230.44.14'}, {u'start': u'10.230...'JL-B01-POD1-CORE-LB-7'}} wrapper /usr/lib/python2.7/site-packages/oslo_log/helpers.py:66`,

			// member
			`2020-10-05 12:14:41.917 295263 DEBUG f5_openstack_agent.lbaasv2.drivers.bigip.agent_manager [req-8f058904-e3f8-401b-b637-97cb5b46f7eb 009ac6496334436a8eba8daa510ef659 62c38230485b4794a8eedece5dac9192 - - -] f5_openstack_agent.lbaasv2.drivers.bigip.agent_manager.LbaasAgentManager method create_member called with arguments (<neutron_lib.context.Context object at 0x7648a50>,) {u'member': {u'name': u'', u'weight': 5, u'admin_state_up': True, u'subnet_id': u'5ee954be-8a76-4e42-b7a9-13a08e5330ce', u'tenant_id': u'62c38230485b4794a8eedece5dac9192', u'provisioning_status': u'PENDING_CREATE', u'pool_id': u'7aabf08d-70aa-4df8-a26f-fde15893b90f', u'address': u'10.230.3.17', u'protocol_port': 39161, u'id': u'43b2c465-d82d-4a5f-951d-8f30837be3f2', u'operating_status': u'OFFLINE'}, u'service': {u'subnets': {u'5ee954be-8a76-4e42-b7a9-13a08e5330ce': {u'updated_at': ...emetal'}, u'operating_status': u'ONLINE', u'name': u'JL-B01-POD1-CORE-LB-2'}}} wrapper /usr/lib/python2.7/site-packages/oslo_log/helpers.py:66`,
		},

		"rest_call_bigip": {
			// get
			`2020-10-05 10:19:16.317 295263 DEBUG root [req-92db71fb-8513-431b-ac79-5423a749b6d7 009ac6496334436a8eba8daa510ef659 62c38230485b4794a8eedece5dac9192 - - -] get WITH uri: https://10.216.177.8:443/mgmt/tm/sys/folder/~CORE_62c38230485b4794a8eedece5dac9192 AND suffix:  AND kwargs: {} wrapper /usr/lib/python2.7/site-packages/icontrol/session.py:257`,

			// post
			`2020-10-28 16:17:55.281 151202 DEBUG root [req-3b85ab54-c3c6-4032-9ff7-6a56233d27d7 a975df1b007d413c8ebc2e90d46232cf 94f2338bf383405db151c4784c0e358c - - -] post WITH uri: https://10.250.2.211:443/mgmt/tm/ltm/pool/~CORE_94f2338bf383405db151c4784c0e358c~CORE_b9453b10-fe39-4667-88ea-172ba8eac39c/members/ AND suffix:  AND kwargs: {'json': {'partition': u'CORE_94f2338bf383405db151c4784c0e358c', 'session': 'user-enabled', 'ratio': 1, 'name': u'4.10.10.15:80', 'address': u'4.10.10.15'}} wrapper /usr/lib/python2.7/site-packages/icontrol/session.py:257`,
		},

		"rest_bigip_response": {
			`2020-10-28 16:17:55.280 151202 DEBUG root [req-3b85ab54-c3c6-4032-9ff7-6a56233d27d7 a975df1b007d413c8ebc2e90d46232cf 94f2338bf383405db151c4784c0e358c - - -] RESPONSE::STATUS: 200 Content-Type: application/json; charset=UTF-8 Content-Encoding: None`,
		},

		"update_loadbalancer_status": {
			// loadbalancer
			`2020-10-05 10:19:18.411 295263 DEBUG f5_openstack_agent.lbaasv2.drivers.bigip.plugin_rpc [req-92db71fb-8513-431b-ac79-5423a749b6d7 009ac6496334436a8eba8daa510ef659 62c38230485b4794a8eedece5dac9192 - - -] f5_openstack_agent.lbaasv2.drivers.bigip.plugin_rpc.LBaaSv2PluginRPC method update_loadbalancer_status called with arguments (u'e2d277f7-eca2-46a4-bf2c-655856fd8733', 'ACTIVE', 'ONLINE', u'JL-B01-POD1-CORE-LB-7') {} wrapper /usr/lib/python2.7/site-packages/oslo_log/helpers.py:66`,
		},

		// "test_basic_pattern":[]string{
		// 		// loadbalancer
		// 		`LoadBalancerManager`,
		// 	},
	}
}
