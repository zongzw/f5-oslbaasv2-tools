package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// StringArray array of string
type StringArray []string

// ArrayOps array's operation
type ArrayOps interface {
	IndexOf(item interface{})
}

// NeutronResponse represent neutron command's response
type NeutronResponse struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	ProvisioningStatus string `json:"provisioning_status"`
}

// CommandContext saved command information and analytics data.
type CommandContext struct {
	Seq           int           `json:"seqnum"`
	Command       string        `json:"command"`
	ObjectID      string        `json:"object_id"`
	RawOut        string        `json:"output"`
	Err           string        `json:"error"`
	CLIRequests   []string      `json:"cli_requests"`
	ExitCode      int           `json:"exitcode"`
	Duration      time.Duration `json:"duration"`
	ResourceType  string        `json:"resource_type"`
	OperationType string        `json:"operation_type"`
	LoadBalancer  string        `json:"loadbalancer"`
}

var (
	logger  = log.New(os.Stdout, "", log.LstdFlags)
	usage   = fmt.Sprintf("Usage: \n\n    %s [command arguments] -- <neutron command and arguments>[ ++ variable-definition]\n\n", os.Args[0])
	example = fmt.Sprintf("Example:\n\n    %s --output-filepath /dev/stdout \\\n    "+
		"-- loadbalancer-create --name lb%s %s \\\n    ++ x:1-5 y:private-subnet,public-subnet\n\n", os.Args[0], "{x}", "{y}")
	varRegexp      = regexp.MustCompile(`%\{[a-zA-Z_][a-zA-Z0-9_]*\}`)
	cliTraceRegexp = regexp.MustCompile(`\w+ call to .* used request id req-.*`)

	cmdList = []string{}

	outputFilePath string
	checkLB        string
	outputFile     *os.File
	mysqluri       string
	dbConn         *gorm.DB = nil

	cmdResults = []*CommandContext{}
	cmdPrefix  = "neutron --debug "

	chsig = make(chan os.Signal)

	maxCheckTimes = 64
)

func main() {

	HandleArguments()

	signal.Notify(chsig, syscall.SIGINT, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGKILL)
	go signalProcess()

	if !strings.Contains(strings.Join(os.Environ(), ","), "OS_USERNAME=") {
		fmt.Println("No OS_USERNAME environment found. Execute `source <path/to/openrc>` first!")
		os.Exit(1)
	}

	neutron, err := exec.LookPath("neutron")
	if err != nil {
		logger.Fatal(err)
	}
	logger.Printf("%20s: %s", "Neutron Command", neutron)

	ExecuteNeutronCommands()
	WriteResult()
	PrintReport()
}

func signalProcess() {
	<-chsig
	logger.Printf("Signal received, quit. Partial results are output to %s", outputFilePath)
	WriteResult()
	PrintReport()

	os.Exit(0)
}

// WriteResult to files
func WriteResult() {
	defer outputFile.Close()

	jd, _ := json.MarshalIndent(cmdResults, "", "  ")
	n, e := outputFile.WriteString(string(jd))
	logger.Printf("Writen executions to file %s: data-len:%d", outputFilePath, n)
	if e != nil {
		logger.Fatalf("Error happens while writing: %s", e.Error())
	}
}

// PrintReport print a summary to the executions.
func PrintReport() {

	fmt.Println()
	fmt.Println("---------------------- Execution Report ----------------------")
	fmt.Println()
	for _, n := range cmdResults {
		fmt.Printf("%d: %s | Exited: %d | duration: %d ms\n",
			n.Seq, n.Command, n.ExitCode, n.Duration.Milliseconds())
	}
	fmt.Println()
	fmt.Println("Failed Command List:")
	for _, n := range cmdResults {
		if n.ExitCode != 0 {
			fmt.Println(n.Command)
		}
	}
	fmt.Println()
	fmt.Println("-----------------------Execution Report End ---------------------")
	fmt.Println()
}

// Execute will execute neutron lbaas-xxxx command and fill with result.
func (cmdctx *CommandContext) Execute() {
	cmdArgs := strings.Split(cmdctx.Command, " ")
	cmdArgs = append(cmdArgs, "--format", "json")
	var out, err bytes.Buffer

	timeoutctx, cancel := context.WithTimeout(context.Background(), time.Duration(30)*time.Minute)
	defer cancel()
	c := exec.CommandContext(timeoutctx, cmdArgs[0], cmdArgs[1:]...)

	c.Env = os.Environ()
	c.Stdout = &out
	c.Stderr = &err

	fs := time.Now()
	e := c.Start()
	if e != nil {
		err.WriteString(e.Error())
	} else {
		e = c.Wait()
		if e != nil {
			err.WriteString(e.Error())
			cmdctx.Err = err.String()
		} else {
			cmdctx.RawOut = out.String()
			var resp NeutronResponse
			if json.Unmarshal(out.Bytes(), &resp) == nil {
				cmdctx.ObjectID = resp.ID
			}
		}
	}
	cmdctx.CLIRequests = cliTraceRegexp.FindAllString(err.String(), -1)

	fe := time.Now()
	cmdctx.ExitCode = c.ProcessState.ExitCode()
	cmdctx.Duration = fe.Sub(fs)
}

// NewCommandContext ...
func NewCommandContext(commandline string) *CommandContext {
	lbAndCmd := strings.Split(commandline, "|")

	fullCmd := fmt.Sprintf("%s%s", cmdPrefix, lbAndCmd[1])

	cmdctx := CommandContext{
		Command: fullCmd,
	}
	cmdctx.LoadBalancer = lbAndCmd[0]

	args := strings.Split(cmdctx.Command, " ")
	subcmd := ""
	for _, arg := range args {
		if strings.HasPrefix(arg, "lbaas-") {
			subcmd = arg
			break
		}
	}
	subs := strings.Split(subcmd, "-")
	cmdctx.ResourceType = subs[1]
	cmdctx.OperationType = subs[2]

	return &cmdctx
}

// ExecuteNeutronCommands Execute the generated commands analyze result.
func ExecuteNeutronCommands() {
	for i, n := range cmdList {
		cmdctx := NewCommandContext(n)
		cmdctx.Seq = i + 1

		logger.Println()
		logger.Printf("Command(%d/%d): Prepare to run '%s'", i+1, len(cmdList), cmdctx.Command)
		if err := cmdctx.WaitForReady(); err != nil {
			logger.Printf("Command(%d/%d): Not ready to run this command: %s", i+1, len(cmdList), err.Error())
			continue
		}

		logger.Printf("Command(%d/%d): Start '%s'", i+1, len(cmdList), cmdctx.Command)
		cmdctx.Execute()

		logger.Printf("Command(%d/%d): exits with: %d, executing time: %d ms",
			cmdctx.Seq, len(cmdList), cmdctx.ExitCode, cmdctx.Duration.Milliseconds())
		time.Sleep(time.Duration(1) * time.Second)

		// check the command execution.
		if cmdctx.ExitCode == 0 {
			cmdctx.WaitForDone()
		} else {
			logger.Printf("Command(%d/%d): Error output: %s", cmdctx.Seq, len(cmdList), cmdctx.Err)
		}
		cmdResults = append(cmdResults, cmdctx)
	}
}

// DBProvisioningStatusOf get object provisioning status
func DBProvisioningStatusOf(objectType string, objectIDName string, isID bool) (string, error) {
	table := "unknown"
	switch objectType {
	case "loadbalancer":
		table = "lbaas_loadbalancers"
	case "pool":
		table = "lbaas_pools"
	case "listener":
		table = "lbaas_listeners"
	case "healthmonitor":
		table = "lbaas_healthmonitors"
	case "member":
		table = "lbaas_members"
	case "l7policy":
		table = "lbaas_l7policies"
	}

	entries := []NeutronResponse{}
	tag := "id"
	if !isID {
		tag = "name"
	}
	rlt := dbConn.Table(table).Where(fmt.Sprintf("%s = ?", tag), objectIDName).Find(&entries)
	if rlt.Error != nil {
		return "", rlt.Error
	}
	if rlt.RowsAffected != 1 {
		return "", fmt.Errorf("%s %s has %d records", objectType, objectIDName, rlt.RowsAffected)
	}

	return entries[0].ProvisioningStatus, nil
}

// LBStatusFromCmd ...
func LBStatusFromCmd(lbIDName string) (string, error) {
	chkctx := CommandContext{
		Command: fmt.Sprintf("neutron lbaas-loadbalancer-show %s", lbIDName),
	}
	chkctx.Execute()
	if chkctx.ExitCode != 0 {
		return "", fmt.Errorf("%s", chkctx.Err)
	}

	var resp NeutronResponse
	_ = json.Unmarshal([]byte(chkctx.RawOut), &resp)

	return resp.ProvisioningStatus, nil
}

// LBStatusFromDB ...
func LBStatusFromDB(lbIDname string) (string, error) {
	isID, _ := regexp.MatchString(`[0-9a-f\-]{36}`, lbIDname)
	return DBProvisioningStatusOf("loadbalancer", lbIDname, isID)
}

// WaitForReady check the loadbalancer is not pending.
func (cmdctx *CommandContext) WaitForReady() error {

	logPrefix := fmt.Sprintf("Command(%d/%d):", cmdctx.Seq, len(cmdList))

	if cmdctx.OperationType == "show" || cmdctx.OperationType == "list" ||
		(cmdctx.ResourceType == "loadbalancer" && cmdctx.OperationType == "create") {
		return nil
	}

	logger.Printf("%s Confirm %s is not pending", logPrefix, cmdctx.LoadBalancer)

	maxErrTries := 3
	errTried := 0
	for retries := maxCheckTimes; retries > 0; retries-- {
		var status string
		var err error
		if dbConn != nil {
			status, err = LBStatusFromDB(cmdctx.LoadBalancer)
		} else {
			status, err = LBStatusFromCmd(cmdctx.LoadBalancer)
		}

		if err != nil {
			logger.Printf("%s Checking loadbalancer(%s) status failed: %s",
				logPrefix, cmdctx.LoadBalancer, err.Error())
			errTried++
			if errTried >= maxErrTries {
				return fmt.Errorf("Loadbalancer %s status check fails for %d times, last failure: %s",
					cmdctx.LoadBalancer, maxErrTries, err.Error())
			}
		} else {
			errTried = 0
		}

		logger.Printf("%s Checked loadbalancer %s status %s",
			logPrefix, cmdctx.LoadBalancer, status)

		if strings.HasPrefix(status, "PENDING_") {
			time.Sleep(time.Duration(1) * time.Second)
			continue
		} else {
			return nil
		}
	}

	return fmt.Errorf("Loadbalancer %s is still PENDING after %d times' check", cmdctx.LoadBalancer, maxCheckTimes)
}

// WaitForDone ...
func (cmdctx *CommandContext) WaitForDone() (bool, error) {
	fs := time.Now()
	defer func() {
		fe := time.Now()
		logger.Printf("Command(%d/%d): Checked time: %d ms", cmdctx.Seq, len(cmdList), fe.Sub(fs).Milliseconds())
	}()

	if cmdctx.OperationType == "create" || cmdctx.OperationType == "update" || cmdctx.OperationType == "delete" {
		if cmdctx.LoadBalancer == "" {
			logger.Printf("Command(%d/%d): No loadbalancer appointed, no check to do.", cmdctx.Seq, len(cmdList))
			return true, nil
		} else if cmdctx.ResourceType == "loadbalancer" && cmdctx.OperationType == "delete" {
			logger.Printf("Command(%d/%d): Loadbalancer deleted, no check to do.", cmdctx.Seq, len(cmdList))
			return true, nil
		} else {
			logger.Printf("Command(%d/%d): Check loadbalancer %s status", cmdctx.Seq, len(cmdList), cmdctx.LoadBalancer)
			for maxTries := maxCheckTimes; maxTries > 0; maxTries-- {
				var status string
				var err error

				// Check created object's status
				if dbConn != nil && cmdctx.ObjectID != "" {
					status, err = DBProvisioningStatusOf(cmdctx.ResourceType, cmdctx.ObjectID, true)
					if err != nil {
						logger.Printf("Command(%d/%d): Failed to fetch object %s status: %s",
							cmdctx.Seq, len(cmdList), cmdctx.ObjectID, err.Error())
						break
					} else if strings.HasPrefix(status, "PENDING_") {
						time.Sleep(time.Duration(1) * time.Second)
						continue
					}
				}

				// Check belonged loadbalancer's status
				if dbConn != nil {
					status, err = LBStatusFromDB(cmdctx.LoadBalancer)
				} else {
					status, err = LBStatusFromCmd(cmdctx.LoadBalancer)
				}
				if err != nil {
					logger.Printf("Command(%d/%d): Checked loadbalancer %s Failed: %s",
						cmdctx.Seq, len(cmdList), cmdctx.LoadBalancer, err.Error())
					break
				}

				logger.Printf("Command(%d/%d): Loadbalancer %s staus is %s",
					cmdctx.Seq, len(cmdList), cmdctx.LoadBalancer, status)
				if strings.HasPrefix(status, "PENDING_") {
					time.Sleep(time.Duration(1) * time.Second)
					continue
				} else {
					return true, nil
				}
			}
			return false, fmt.Errorf("LB: %s left PENDING", cmdctx.LoadBalancer)
		}
	} else { // 'show' 'list' no need to check
		return true, nil
	}
}

// HandleArguments handle user's input.
func HandleArguments() {
	flag.StringVar(&outputFilePath, "output-filepath", "/dev/stdout", "output the result")
	flag.IntVar(&maxCheckTimes, "max-check-times", maxCheckTimes, "The max times for checking loadbalancer is ready for next step.")
	flag.StringVar(&checkLB, "check-lb", "", "the loadbalancer name or id for checking execution status.")
	flag.StringVar(&mysqluri, "mysql-uri", "", "database connection string")

	flag.Usage = PrintUsage
	flag.Parse()

	if mysqluri != "" {
		// mysql conn string example: neutron:abd2aebadeff3e32@tcp(1.2.3.4:3306)/ovs_neutron
		matched, _ := regexp.MatchString(`\w+:\w+@tcp\([0-9\.]+:\d+\)/\w+`, mysqluri)
		if !matched {
			logger.Fatalf("Invalid mysql uri provided: %s", mysqluri)
		}
		conn, err := gorm.Open(mysql.Open(mysqluri), &gorm.Config{})
		if err != nil {
			logger.Fatal(err)
		}
		dbConn = conn
		logger.Printf("%20s: %s", "MySQL URI", mysqluri)
	}

	of, e := os.OpenFile(outputFilePath, os.O_CREATE|os.O_RDWR|os.O_APPEND, os.ModeAppend|os.ModePerm)
	if e != nil {
		logger.Fatalf("Failed to open file %s for writing.", e.Error())
	}
	outputFile = of
	logger.Printf("%20s: %s", "Output File Path", outputFilePath)

	neutronArgsIndex := StringArray(os.Args).IndexOf("--")
	if neutronArgsIndex == -1 {
		logger.Fatal(usage)
	}

	variableArgsIndex := StringArray(os.Args).IndexOf("++")
	if variableArgsIndex == -1 {
		variableArgsIndex = len(os.Args)
	}

	neutronCmdArgs := strings.Join(os.Args[neutronArgsIndex+1:variableArgsIndex], " ")
	neutronCmdArgs = checkLB + "|" + neutronCmdArgs
	logger.Printf("%20s: %s", "Command Template", neutronCmdArgs)

	variables := map[string]StringArray{}

	varStart := false

	for _, n := range os.Args[neutronArgsIndex+1:] {
		if n == "++" {
			varStart = true
			continue
		}

		if !varStart {
			matches := varRegexp.FindAllString(n, -1)
			for _, m := range matches {
				// logger.Printf("found variable: %s\n", m)
				l := len(m)
				varName := m[2 : l-1]
				variables[varName] = []string{}
			}
		} else {
			for k := range variables {
				if strings.HasPrefix(n, fmt.Sprintf("%s:", k)) {
					kvp := strings.Split(n, ":")
					v := ParseVarValues(strings.Join(kvp[1:], ":"))
					variables[k] = append(variables[k], v...)
				}
			}
		}
	}

	logger.Printf("%20s:", "Variables")
	for k, v := range variables {
		logger.Printf("%30s: %v", k, v)
	}

	ConstructFromTemplate(neutronCmdArgs, variables)

	// Random cmdList order to help reducing objects' waiting time in the same loadbalancer.
	for i := range cmdList {
		r := rand.Int() % len(cmdList)
		t := cmdList[r]
		cmdList[r] = cmdList[i]
		cmdList[i] = t
	}
}

// PrintUsage print the usage
func PrintUsage() {
	fmt.Fprintf(os.Stderr, usage)
	fmt.Fprintf(os.Stderr, example)
	fmt.Fprintf(os.Stderr, "Command Arguments: \n\n")
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, "\n")
}

// ConstructFromTemplate recursively generate the command from templete
func ConstructFromTemplate(template string, variables map[string]StringArray) {
	varInTmp := varRegexp.FindString(template)
	if varInTmp == "" {
		cmdList = append(cmdList, template)
		return
	}
	l := len(varInTmp)
	varName := varInTmp[2 : l-1]

	r := regexp.MustCompile(varInTmp)

	for _, k := range variables[varName] {
		replaced := r.ReplaceAllString(template, k)
		ConstructFromTemplate(replaced, variables)
	}
}

// ParseVarValues parse the value ranges to actual value list
// Supports: '-' num list and ',' list
//		1-5
// 		a,b,c
// 		1-3,4,6-9,a,b,c
func ParseVarValues(v string) []string {
	rlt := []string{}
	ls := strings.Split(v, ",")
	p := regexp.MustCompile(`^\d+\-\d+$`)
	for _, n := range ls {
		matched := p.MatchString(n)
		if matched {
			se := strings.Split(n, "-")
			s, _ := strconv.Atoi(se[0])
			e, _ := strconv.Atoi(se[1])
			for i := s; i <= e; i++ {
				rlt = append(rlt, fmt.Sprintf("%d", i))
			}
		} else {
			rlt = append(rlt, n)
		}
	}
	return rlt
}

// IndexOf Implement the StringArray's IndexOf
func (sa StringArray) IndexOf(item string) int {
	for i, n := range sa {
		if n == item {
			return i
		}
	}
	return -1
}
