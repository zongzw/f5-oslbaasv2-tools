package main

import (
	"strings"
	"fmt"
	"log"
	"flag"
	"os"
	"bufio"
	"time"
	// "encoding/json"
	"path/filepath"
	"sync"
	// "regexp"
	"github.com/trivago/grok"
	// "github.com/google/uuid"
)

type arrayFlags []string

const FK_TIMELAYOUT = "2006-01-02 15:04:05"

var (
	logger = log.New(os.Stdout, "", log.LstdFlags)
	
	pBasicFields = map[string]string{
		"UUID": `[a-z0-9]{8}-([a-z0-9]{4}-){3}[a-z0-9]{12}`,    	// 6245c77d-5017-4657-b35b-7ab1d247112b
		"REQID": `req-%{UUID}`,										// req-8cadad28-8315-45ca-818c-6a229dfb73e1
		"DATETIME": `\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}.\d{3}`,	// 2020-09-27 19:22:54.486
		"MD5": `[0-9a-z]{32}`, 										// 62c38230485b4794a8eedece5dac9192
		"JSON": `\{.*\}`,											// {u'bandwidth_limit_rule': {u'max_kbps': 102400, u'direction': u'egress', u'max_burst_kbps': 102400}}
		"LBTYPE": `(LoadBalancer|Listener|Pool|Member|HealthMonitor)`,
		"LBTYPESTR": `(loadbalancer|listener|pool|member|health_monitor)`,
		"ACTION": `(create|update|delete)`,
	}

	pLBaaSv2 = map[string]string{

		// 2020-09-27 19:22:54.485 68316 DEBUG neutron.api.v2.base 
		// [req-8cadad28-8315-45ca-818c-6a229dfb73e1 009ac6496334436a8eba8daa510ef659 62c38230485b4794a8eedece5dac9192 - default default] Request body: 
		// {u'bandwidth_limit_rule': {u'max_kbps': 102400, u'direction': u'egress', u'max_burst_kbps': 102400}} 
		// prepare_request_body /usr/lib/python2.7/site-packages/neutron/api/v2/base.py:713
		"neutron_api_v2_base": `%{DATETIME:neutron_api_time} .* neutron.api.v2.base \[%{REQID:req_id} .*\] ` +
							   `Request body: %{JSON:request_body} prepare_request_body .*$`,

		// 05neu-core/server.log-1005:2020-10-05 10:20:17.251 117825 DEBUG f5lbaasdriver.v2.bigip.driver_v2 
		// [req-92db71fb-8513-431b-ac79-5423a749b6d7 009ac6496334436a8eba8daa510ef659 62c38230485b4794a8eedece5dac9192 - default default] 
		// f5lbaasdriver.v2.bigip.driver_v2.LoadBalancerManager method create called with arguments (<neutron_lib.context.Context object at 0x284cb250>, 
		// <neutron_lbaas.services.loadbalancer.data_models.LoadBalancer object at 0xdb44250>) {} 
		// wrapper /usr/lib/python2.7/site-packages/oslo_log/helpers.py:66
		"call_f5driver": 
			`%{DATETIME:call_f5driver_time} .* f5lbaasdriver.v2.bigip.driver_v2 \[%{REQID:req_id} .*\] ` +
			`f5lbaasdriver.v2.bigip.driver_v2.%{LBTYPE:object_type}Manager method %{ACTION:operation_type} called with .*$`,
		
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
		"rpc_f5agent": 
			`%{DATETIME:rpc_f5agent_time} .* f5lbaasdriver.v2.bigip.agent_rpc \[%{REQID:req_id} .*\] ` +
			`f5lbaasdriver.v2.bigip.agent_rpc.LBaaSv2AgentRPC method %{ACTION}_%{LBTYPESTR} called with arguments ` +
			`.*? 'id': '%{UUID:object_id}'.*`,

		// 2020-10-05 10:19:16.315 295263 DEBUG f5_openstack_agent.lbaasv2.drivers.bigip.agent_manager 
		// [req-92db71fb-8513-431b-ac79-5423a749b6d7 009ac6496334436a8eba8daa510ef659 62c38230485b4794a8eedece5dac9192 - - -] 
		// f5_openstack_agent.lbaasv2.drivers.bigip.agent_manager.LbaasAgentManager method create_loadbalancer called with arguments
		// ...
		// 7'}} wrapper /usr/lib/python2.7/site-packages/oslo_log/helpers.py:66
		"call_f5agent": 
			`%{DATETIME:call_f5agent_time} .* f5_openstack_agent.lbaasv2.drivers.bigip.agent_manager \[%{REQID:req_id} .*\] ` +
			`f5_openstack_agent.lbaasv2.drivers.bigip.agent_manager.LbaasAgentManager method %{ACTION}_%{LBTYPESTR} ` +
			`called with arguments .*`,

		// 2020-10-05 10:19:16.317 295263 DEBUG root [req-92db71fb-8513-431b-ac79-5423a749b6d7 009ac6496334436a8eba8daa510ef659 
		// 62c38230485b4794a8eedece5dac9192 - - -] get WITH uri: https://10.216.177.8:443/mgmt/tm/sys/folder/~CORE_62c38230485b4794a8eedece5dac9192 AND 
		// suffix:  AND kwargs: {} wrapper /usr/lib/python2.7/site-packages/icontrol/session.py:257
		"rest_call_bigip": 
			`%{DATETIME:call_bigip_time} .* \[%{REQID:req_id} .*\] get WITH uri: .*icontrol/session.py.*`,

		// 2020-10-05 10:19:18.411 295263 DEBUG f5_openstack_agent.lbaasv2.drivers.bigip.plugin_rpc 
		// [req-92db71fb-8513-431b-ac79-5423a749b6d7 009ac6496334436a8eba8daa510ef659 62c38230485b4794a8eedece5dac9192 - - -] 
		// f5_openstack_agent.lbaasv2.drivers.bigip.plugin_rpc.LBaaSv2PluginRPC method update_loadbalancer_status called with arguments 
		// (u'e2d277f7-eca2-46a4-bf2c-655856fd8733', 'ACTIVE', 'ONLINE', u'JL-B01-POD1-CORE-LB-7') {} wrapper 
		// /usr/lib/python2.7/site-packages/oslo_log/helpers.py:66
		"update_loadbalancer_status": 
			`%{DATETIME:update_status_time} .* f5_openstack_agent.lbaasv2.drivers.bigip.plugin_rpc \[%{REQID:req_id} .*\].* ` +
			`method update_loadbalancer_status called with arguments.*`,
		
		// "test_basic_pattern":
		// 	`%{LBTYPE:object_type}`,
	}

	linesBuffSize = 50
	linesBuff = make([]string, linesBuffSize)

	result = map[string]map[string]string{}

	rltLock = &sync.Mutex{}
	bufLock = &sync.Mutex{}

	readDone = false

	nThrParse = 100
	nThrReadMax = 10
	
	chThrFiles = make(chan bool, nThrReadMax)
	
	wgParse = sync.WaitGroup{}
	wgRead = sync.WaitGroup{}

	debugSize = 50000
)

func main() {

	var logpaths arrayFlags
	var output_filepath string
	// var output_ts string
	var t bool

	flag.Var(&logpaths, "logpath", 
		"The log path for analytics. It can be used multiple times. " +
		"Path regex pattern can be used WITHIN \"\".\n" +
		"e.g: --logpath /path/to/f5-openstack-agent.log " +
		"--logpath \"/var/log/neutron/f5-openstack-*.log\" " +
		"--logpath /var/log/neutron/server\\*.log")
	flag.StringVar(&output_filepath, "output-filepath", "", 
		"Output the result to file, e.g: /path/to/result.csv")
	// TODO: output result to f5 telemetry analytics.
	// flag.StringVar(&output_ts, "output-ts", "./result.json", 
	// 	"Output the result to f5-telemetry-analytics. e.g: http://1.1.1.1:200002")
	flag.BoolVar(&t, "test", false, "Program self test option..")
	flag.Parse()

	if t {
		TestProg()
		return
	}

	fileHandlers, err := HandleArguments(logpaths, output_filepath)
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

	for i:=0; i<nThrParse; i++ {
		wgParse.Add(1)
		go Parse(g)
	}
	
	for _, f := range(fileHandlers) {
		wgRead.Add(1)
		go Read(f)
	}

	wgRead.Wait()
	readDone = true
	wgParse.Wait()

	CalculateDuration(result)

	OutputResult(output_filepath, result)
}

func Parse(g *grok.Grok) {
	defer wgParse.Done()

	for true {
		if len(linesBuff) == 0 {
			if readDone { break }
			time.Sleep(time.Duration(5) * time.Microsecond)
			continue
		}

		t := ""
		bufLock.Lock()
		if len(linesBuff) != 0 {
			t = linesBuff[0]
			linesBuff = linesBuff[1:]
		}
		bufLock.Unlock()

		if t != "" {
			for k, _ := range pLBaaSv2 {
				Parse2Result(g, k, t)
			}
		}
	}
}

func Parse2Result(g *grok.Grok, k string, text string) {
	values, err := g.ParseString(fmt.Sprintf("%%{%s}", k), text)
	if err != nil { logger.Fatal(err) }
	if len(values) == 0 { return }

	rltLock.Lock()
	if _, ok := values["req_id"]; !ok {
		logger.Fatalf("Abnormal thing happens. No req_id matched: pattern key: %s, log: %s\n", k, text)
	}

	req_id := values["req_id"]
	if _, ok := result[req_id]; !ok {
		result[req_id] = map[string]string{}
	}
	for k, v := range values {
		result[req_id][k] = v
	}
	rltLock.Unlock()
}

func Read(f *os.File) {
	chThrFiles <- true
	defer func(){
		<- chThrFiles
		wgRead.Done()
	}()
	log.Printf("Start to reading %s\n", f.Name())
	
	scanner := bufio.NewScanner(f)
	maxCapacity := 512 * 1024  // default max size 64*1024
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	fs := time.Now().UnixNano()
	lines := 0

	for scanner.Scan() {
		lines += 1
		bufLock.Lock()
		linesBuff = append(linesBuff, scanner.Text())
		if lines % debugSize == 0 {
			logger.Printf("%s, read lines %d, len of linesBuff: %d\n", f.Name(), lines, len(linesBuff))
		}
		bufLock.Unlock()

		if len(linesBuff) > linesBuffSize {
			time.Sleep(time.Duration(5) * time.Microsecond)
		}
	}

	fe := time.Now().UnixNano()
	if err := scanner.Err(); err != nil {
		logger.Printf("Error happens at %s line %d\n", f.Name(), lines)
		logger.Fatal(err)
	} else {
		logger.Printf("Read from file %s, lines: %d, total time: %v ms \n", 
			f.Name(), lines, (fe - fs)/1e6)
	}
}

func OutputResult(filepath string, result map[string]map[string]string) {
	fw, e := os.Create(filepath)
	if e != nil {
		logger.Fatal(e)
	}
	defer fw.Close()

	title_line := []string{
		"req_id", "object_id", "object_type", "operation_type", "request_body", 
		"neutron_api_time", "call_f5driver_time", "rpc_f5agent_time", "call_f5agent_time", "call_bigip_time", "update_status_time", 
		"dur_neu_drv", "dur_drv_rpc", "dur_rpc_agt", "dur_agt_upd", "total", 
	}

	_, e = fw.WriteString(strings.Join(title_line, ",") + "\n")
	if e != nil {
		logger.Fatal(e)
	}

	for _, v := range result {
		a := []string{}
		for _, n := range title_line {
			if strings.HasSuffix(n, "_time") {
				a = append(a, fmt.Sprintf("\"T%s\"", v[n]))
			} else {
				a = append(a, fmt.Sprintf("\"%s\"", v[n]))
			}
			
		}
		line := strings.Join(a, ",") + "\n"

		_, e := fw.WriteString(line)
		if e != nil {
			logger.Fatal(e)
		}
	}
}

func MakeGrok() (*grok.Grok, error) {
	pattern :=map[string]string {}

	for k, v := range pBasicFields {
		pattern[k] = v
	}
	for k, v := range pLBaaSv2 {
		pattern[k] = v
	}

	g, e := grok.New(grok.Config{
		NamedCapturesOnly: true,
		Patterns: pattern,
	})
	return g, e
}

func HandleArguments(logpaths []string, output_filepath string) ([]*os.File, error){

	// handle output file for result.
	if output_filepath == "" {
		return nil, fmt.Errorf("--output-filepath should be appointed.")
	}
	dir, err := filepath.Abs(filepath.Dir(output_filepath))
	if err != nil {
		return nil, err
	} else {
		_, err := os.Stat(dir)
		if os.IsNotExist(err) {
			return nil, err
		}
	}
	_, err = os.Stat(output_filepath)
	if err == nil {
		return nil, fmt.Errorf("File %s already exists.", output_filepath)
	}

	var fileHandlers []*os.File

	// parse regex paths
	paths :=[]string{}
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
		return nil, fmt.Errorf("Invalid file path detected, cannot to continue.\n")
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
		return nil, fmt.Errorf("Invalid paths found while getting the absolute path. Cannot continue.\n")
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
		f, err := os.Stat(string(p))
		if err == nil {
		}

		if os.IsNotExist(err) {
			logger.Println(err.Error())
			pathOK = false
		}

		if f.IsDir() {
			logger.Printf("%s should be a file, not a directory.\n", f.Name())
			pathOK = false
		}

		fr, err := os.Open(p)
		if err != nil {
			return nil, err
		}
		fileHandlers = append(fileHandlers, fr)
	}
	if !pathOK {
		return nil, fmt.Errorf("Invalid path(s) provided.")
	}

	return fileHandlers, nil
}

func FKTheTime(datm string) time.Time {
	dt, _ := time.Parse(FK_TIMELAYOUT, datm)
	return dt;
}

func CalculateDuration(result map[string]map[string]string) {
	for _, r := range result {
		tNeutron := FKTheTime(r["neutron_api_time"])
		tDriver := FKTheTime(r["call_f5driver_time"])
		tMQ := FKTheTime(r["rpc_f5agent_time"])
		tAgent := FKTheTime(r["call_f5agent_time"])
		tUpdate := FKTheTime(r["update_status_time"])
		// tBIGIP := FKTheTime(r["call_bigip_time"])
		r["dur_neu_drv"] = fmt.Sprintf("%v", tDriver.Sub(tNeutron))
		r["dur_drv_rpc"] = fmt.Sprintf("%v", tMQ.Sub(tDriver))
		r["dur_rpc_agt"] = fmt.Sprintf("%v", tAgent.Sub(tMQ))
		r["dur_agt_upd"] = fmt.Sprintf("%v", tUpdate.Sub(tAgent))
		r["total"] = fmt.Sprintf("%v", tUpdate.Sub(tNeutron))
	}
}

func (i *arrayFlags) String() string {
	return fmt.Sprint(*i)
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func IsContains(items []string, item string) bool {
	for _, n := range items {
		if n == item {
			return true
		}
	}
	return false
}

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

func TestParse(k string, v string, g *grok.Grok) (map[string]string, error) {
	return g.ParseString(fmt.Sprintf("%%{%s}", k), v)
}

func TestCases() map[string][]string {
	return map[string][]string{
		"neutron_api_v2_base":[]string{
				// loadbalancer
				`2020-10-05 10:20:15.791 117825 DEBUG neutron.api.v2.base [req-92db71fb-8513-431b-ac79-5423a749b6d7 009ac6496334436a8eba8daa510ef659 62c38230485b4794a8eedece5dac9192 - default default] Request body: {u'loadbalancer': {u'vip_subnet_id': u'd79ef712-c1e3-4860-9343-d1702b9976aa', u'provider': u'core', u'name': u'JL-B01-POD1-CORE-LB-7', u'admin_state_up': True}} prepare_request_body /usr/lib/python2.7/site-packages/neutron/api/v2/base.py:713`,	
				// member
				`2020-10-05 14:50:24.795 117812 DEBUG neutron.api.v2.base [req-be08ea84-f721-46da-b24e-6e2c249af84e 009ac6496334436a8eba8daa510ef659 62c38230485b4794a8eedece5dac9192 - default default] Request body: {u'member': {u'subnet_id': u'5ee954be-8a76-4e42-b7a9-13a08e5330ce', u'address': u'10.230.3.39', u'protocol_port': 39130, u'weight': 5, u'admin_state_up': True}} prepare_request_body /usr/lib/python2.7/site-packages/neutron/api/v2/base.py:713`,
			},

		"call_f5driver": []string{
				// loadbalancer
				`2020-10-05 10:20:17.251 117825 DEBUG f5lbaasdriver.v2.bigip.driver_v2 [req-92db71fb-8513-431b-ac79-5423a749b6d7 009ac6496334436a8eba8daa510ef659 62c38230485b4794a8eedece5dac9192 - default default] f5lbaasdriver.v2.bigip.driver_v2.LoadBalancerManager method create called with arguments (<neutron_lib.context.Context object at 0x284cb250>, <neutron_lbaas.services.loadbalancer.data_models.LoadBalancer object at 0xdb44250>) {} wrapper /usr/lib/python2.7/site-packages/oslo_log/helpers.py:66`,
				// member
				`2020-10-05 14:50:28.214 117812 DEBUG f5lbaasdriver.v2.bigip.driver_v2 [req-be08ea84-f721-46da-b24e-6e2c249af84e 009ac6496334436a8eba8daa510ef659 62c38230485b4794a8eedece5dac9192 - default default] f5lbaasdriver.v2.bigip.driver_v2.MemberManager method create called with arguments (<neutron_lib.context.Context object at 0x1310cc90>, <neutron_lbaas.services.loadbalancer.data_models.Member object at 0x286ed750>) {} wrapper /usr/lib/python2.7/site-packages/oslo_log/helpers.py:66`,
			},

		"rpc_f5agent": []string{
				// loadbalancer
				`2020-10-05 10:20:27.176 117825 DEBUG f5lbaasdriver.v2.bigip.agent_rpc [req-92db71fb-8513-431b-ac79-5423a749b6d7 009ac6496334436a8eba8daa510ef659 62c38230485b4794a8eedece5dac9192 - default default] f5lbaasdriver.v2.bigip.agent_rpc.LBaaSv2AgentRPC method create_loadbalancer called with arguments (<neutron_lib.context.Context object at 0x284cb250>, {'availability_zone_hints': [], 'description': '', 'admin_state_up': True, 'tenant_id': '62c38230485b4794a8eedece5dac9192', 'provisioning_status': 'PENDING_CREATE', 'listeners': [], 'vip_subnet_id': 'd79ef712-c1e3-4860-9343-d1702b9976aa', 'vip_address': '10.230.44.15', 'vip_port_id': '5bcbe2d7-994f-40de-87ab-07aa632f0133', 'provider': None, 'pools': [], 'id': 'e2d277f7-eca2-46a4-bf2c-655856fd8733', 'operating_status': 'OFFLINE', 'name': 'JL-B01-POD1-CORE-LB-7'}, {'subnets': {u'd79ef71...OD1_CORE3') {} wrapper /usr/lib/python2.7/site-packages/oslo_log/helpers.py:66`,

				// member
				`2020-10-05 14:51:54.445 117812 DEBUG f5lbaasdriver.v2.bigip.agent_rpc [req-be08ea84-f721-46da-b24e-6e2c249af84e 009ac6496334436a8eba8daa510ef659 62c38230485b4794a8eedece5dac9192 - default default] f5lbaasdriver.v2.bigip.agent_rpc.LBaaSv2AgentRPC method create_member called with arguments (<neutron_lib.context.Context object at 0x1310cc90>, {'name': '', 'weight': 5, 'provisioning_status': 'PENDING_CREATE', 'subnet_id': '5ee954be-8a76-4e42-b7a9-13a08e5330ce', 'tenant_id': '62c38230485b4794a8eedece5dac9192', 'admin_state_up': True, 'pool_id': '100858a1-8ba9-496c-9cb4-7d1143431ce8', 'address': '10.230.3.39', 'protocol_port': 39130, 'id': '551b7992-273f-4923-94f2-57b12a715c15', 'operating_status': 'OFFLINE'}, {'subne...18273-1f5e-4be2-a263-ce37823a7773', 'operating_status': 'ONLINE', 'name': 'JL-B01-POD1-CORE-LB-1'}}, u'POD1_CORE') {} wrapper /usr/lib/python2.7/site-packages/oslo_log/helpers.py:66`,
			},

		"call_f5agent":[]string{
				// loadbalancer
				`2020-10-05 10:19:16.315 295263 DEBUG f5_openstack_agent.lbaasv2.drivers.bigip.agent_manager [req-92db71fb-8513-431b-ac79-5423a749b6d7 009ac6496334436a8eba8daa510ef659 62c38230485b4794a8eedece5dac9192 - - -] f5_openstack_agent.lbaasv2.drivers.bigip.agent_manager.LbaasAgentManager method create_loadbalancer called with arguments (<neutron_lib.context.Context object at 0x7351290>,) {u'service': {u'subnets': {u'd79ef712-c1e3-4860-9343-d1702b9976aa': {u'updated_at': u'2020-09-25T05:29:56Z', u'ipv6_ra_mode': None, u'allocation_pools': [{u'start': u'10.230.44.2', u'end': u'10.230.44.30'}], u'host_routes': [], u'revision_number': 1, u'ipv6_address_mode': None, u'id': u'd79ef712-c1e3-4860-9343-d1702b9976aa', u'available_ips': [{u'start': u'10.230.44.3', u'end': u'10.230.44.3'}, {u'start': u'10.230.44.10', u'end': u'10.230.44.12'}, {u'start': u'10.230.44.14', u'end': u'10.230.44.14'}, {u'start': u'10.230...'JL-B01-POD1-CORE-LB-7'}} wrapper /usr/lib/python2.7/site-packages/oslo_log/helpers.py:66`,

				// member
				`2020-10-05 12:14:41.917 295263 DEBUG f5_openstack_agent.lbaasv2.drivers.bigip.agent_manager [req-8f058904-e3f8-401b-b637-97cb5b46f7eb 009ac6496334436a8eba8daa510ef659 62c38230485b4794a8eedece5dac9192 - - -] f5_openstack_agent.lbaasv2.drivers.bigip.agent_manager.LbaasAgentManager method create_member called with arguments (<neutron_lib.context.Context object at 0x7648a50>,) {u'member': {u'name': u'', u'weight': 5, u'admin_state_up': True, u'subnet_id': u'5ee954be-8a76-4e42-b7a9-13a08e5330ce', u'tenant_id': u'62c38230485b4794a8eedece5dac9192', u'provisioning_status': u'PENDING_CREATE', u'pool_id': u'7aabf08d-70aa-4df8-a26f-fde15893b90f', u'address': u'10.230.3.17', u'protocol_port': 39161, u'id': u'43b2c465-d82d-4a5f-951d-8f30837be3f2', u'operating_status': u'OFFLINE'}, u'service': {u'subnets': {u'5ee954be-8a76-4e42-b7a9-13a08e5330ce': {u'updated_at': ...emetal'}, u'operating_status': u'ONLINE', u'name': u'JL-B01-POD1-CORE-LB-2'}}} wrapper /usr/lib/python2.7/site-packages/oslo_log/helpers.py:66`,
			},

		"rest_call_bigip":[]string{
				// loadbalancer
				`2020-10-05 10:19:16.317 295263 DEBUG root [req-92db71fb-8513-431b-ac79-5423a749b6d7 009ac6496334436a8eba8daa510ef659 62c38230485b4794a8eedece5dac9192 - - -] get WITH uri: https://10.216.177.8:443/mgmt/tm/sys/folder/~CORE_62c38230485b4794a8eedece5dac9192 AND suffix:  AND kwargs: {} wrapper /usr/lib/python2.7/site-packages/icontrol/session.py:257`,
			},

		"update_loadbalancer_status":[]string{
				// loadbalancer
				`2020-10-05 10:19:18.411 295263 DEBUG f5_openstack_agent.lbaasv2.drivers.bigip.plugin_rpc [req-92db71fb-8513-431b-ac79-5423a749b6d7 009ac6496334436a8eba8daa510ef659 62c38230485b4794a8eedece5dac9192 - - -] f5_openstack_agent.lbaasv2.drivers.bigip.plugin_rpc.LBaaSv2PluginRPC method update_loadbalancer_status called with arguments (u'e2d277f7-eca2-46a4-bf2c-655856fd8733', 'ACTIVE', 'ONLINE', u'JL-B01-POD1-CORE-LB-7') {} wrapper /usr/lib/python2.7/site-packages/oslo_log/helpers.py:66`,
			},

		// "test_basic_pattern":[]string{
		// 		// loadbalancer
		// 		`LoadBalancerManager`,
		// 	},
	}
}
