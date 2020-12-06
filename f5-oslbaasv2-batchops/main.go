package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
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
	ProvisioningStatus string `json:"provisioning_status"`
}

// CommandContext saved command information and analytics data.
type CommandContext struct {
	Seq         int           `json:"seqnum"`
	Command     string        `json:"command"`
	RawOut      string        `json:"output"`
	Err         string        `json:"error"`
	ExitCode    int           `json:"exitcode"`
	CmdDuration time.Duration `json:"cmd_duration"`
	Check       struct {
		Result   string          `json:"check_status"`
		Duration time.Duration   `json:"check_duration"`
		LBIDName string          `json:"check_loadbalancer"`
		ShowResp NeutronResponse `json:"check_lb_result"`
	} `json:"check"`
}

var (
	logger  = log.New(os.Stdout, "", log.LstdFlags)
	usage   = fmt.Sprintf("Usage: \n\n    %s [command arguments] -- <neutron command and arguments>[ ++ variable-definition]\n\n", os.Args[0])
	example = fmt.Sprintf("Example:\n\n    %s --output-filepath /dev/stdout \\\n    "+
		"-- loadbalancer-create --name lb%s %s \\\n    ++ x:1-5 y:private-subnet,public-subnet\n\n", os.Args[0], "{x}", "{y}")
	varRegexp = regexp.MustCompile(`%\{[a-zA-Z_][a-zA-Z0-9_]*\}`)
	cmdList   = []string{}

	output     string
	checkLB    string
	outputFile *os.File

	cmdResults = []CommandContext{}
	cmdPrefix  = "neutron "

	chsig = make(chan os.Signal)

	maxCheckTimes = 64
)

func main() {

	HandleArguments()

	signal.Notify(chsig, syscall.SIGINT, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGKILL)
	go signalProcess()

	if output != "/dev/stdout" {
		of, e := os.OpenFile(output, os.O_CREATE|os.O_RDWR|os.O_APPEND, os.ModeAppend|os.ModePerm)
		if e != nil {
			logger.Fatalf("Failed to open file %s for writing.", e.Error())
		}
		outputFile = of
		defer outputFile.Close()
	}

	if !strings.Contains(strings.Join(os.Environ(), ","), "OS_USERNAME=") {
		fmt.Println("No OS_USERNAME environment found. Execute `source <path/to/openrc>` first!")
		os.Exit(1)
	}

	neutron, err := exec.LookPath("neutron")
	if err != nil {
		logger.Fatal(err)
	}
	logger.Printf("neutron command: %s", neutron)

	RunCmds()
	WriteResult()
	PrintReport()
}

func signalProcess() {
	<-chsig
	WriteResult()
	PrintReport()

	logger.Printf("Signal received, quit. Partial results are output to %s", output)
	os.Exit(0)
}

// WriteResult to files
func WriteResult() {
	jd, _ := json.MarshalIndent(cmdResults, "", "  ")
	if output != "/dev/stdout" {
		n, e := outputFile.WriteString(string(jd))
		logger.Printf("Writen executions to file %s: data-len:%d", output, n)
		if e != nil {
			logger.Fatalf("Error happens while writing: %s", e.Error())
		}
	} else {
		fmt.Printf("%s\n", string(jd))
	}
}

// PrintReport print a summary to the executions.
func PrintReport() {

	fmt.Println()
	fmt.Println("---------------------- Execution Report ----------------------")
	fmt.Println()
	for _, n := range cmdResults {
		fmt.Printf("%d: %s | Exited: %d | Checked: %s | duration: %d ms\n",
			n.Seq, n.Command, n.ExitCode, n.Check.Result, n.Check.Duration.Milliseconds())
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

// RunCmds Execute the generated commands analyze result.
func RunCmds() {
	for i, n := range cmdList {
		lbAndCmd := strings.Split(n, "|")

		fullCmd := fmt.Sprintf("%s%s", cmdPrefix, lbAndCmd[1])

		cmdctx := CommandContext{
			Seq:     i + 1,
			Command: fullCmd,
		}
		cmdctx.Check.LBIDName = lbAndCmd[0]

		logger.Println()
		logger.Printf("Command(%d/%d): Prepare to run '%s'", i+1, len(cmdList), fullCmd)
		if err := WaitForLBToNotPending(&cmdctx); err != nil {
			logger.Printf("Command(%d/%d): Not ready to run  this command: %s", i+1, len(cmdList), err.Error())
			return
		}

		logger.Printf("Command(%d/%d): Start '%s'", i+1, len(cmdList), fullCmd)
		RunCmd(&cmdctx)

		logger.Printf("Command(%d/%d): exits with: %d, executing time: %d ms",
			cmdctx.Seq, len(cmdList), cmdctx.ExitCode, cmdctx.CmdDuration.Milliseconds())
		time.Sleep(time.Duration(1) * time.Second)

		// check the command execution.
		if cmdctx.ExitCode == 0 {
			// Temporarily not check the result.
			//CheckLBStatus(&cmdctx)
		} else {
			logger.Printf("Command(%d/%d): Error output: %s", cmdctx.Seq, len(cmdList), cmdctx.Err)
		}
		cmdResults = append(cmdResults, cmdctx)
	}
}

// WaitForLBToNotPending check the loadbalancer is not pending.
func WaitForLBToNotPending(cmdctx *CommandContext) error {

	logPrefix := fmt.Sprintf("Command(%d/%d):", cmdctx.Seq, len(cmdList))
	args := strings.Split(cmdctx.Command, " ")
	subcmd := ""
	for _, arg := range args {
		if strings.HasPrefix(arg, "lbaas-") {
			subcmd = arg
			break
		}
	}
	subs := strings.Split(subcmd, "-")
	resourceType, operation := subs[1], subs[2]
	if operation == "show" || operation == "list" {
		return nil
	} else if resourceType == "loadbalancer" && operation == "create" {
		return nil
	}

	logger.Printf("%s Confirm %s is not pending", logPrefix, cmdctx.Check.LBIDName)

	chkctx := CommandContext{
		Command: fmt.Sprintf("neutron lbaas-loadbalancer-show %s", cmdctx.Check.LBIDName),
	}

	maxChkTries := maxCheckTimes
	maxErrTries := 3
	chkTried := 0
	errTried := 0
	for ; chkTried < maxChkTries; chkTried++ {
		RunCmd(&chkctx)
		if chkctx.ExitCode != 0 {
			logger.Printf("%s Checking loadbalancer(%s) status failed: %s",
				logPrefix, cmdctx.Check.LBIDName, chkctx.Err)
			errTried++
			if errTried >= maxErrTries {
				return fmt.Errorf("Check loadbalancer %s status for %d times, last failure: %s",
					cmdctx.Check.LBIDName, maxErrTries, chkctx.Err)
			}
		} else {
			errTried = 0
		}

		var resp NeutronResponse
		_ = json.Unmarshal([]byte(chkctx.RawOut), &resp)

		logger.Printf("%s Checked loadbalancer %s status %s",
			logPrefix, cmdctx.Check.LBIDName, resp.ProvisioningStatus)

		if strings.HasPrefix(resp.ProvisioningStatus, "PENDING_") {
			time.Sleep(time.Duration(1) * time.Second)
			continue
		} else {
			return nil
		}
	}

	if chkTried >= maxChkTries {
		return fmt.Errorf("Tried %d times to get LB status, still PENDING", maxChkTries)
	}

	return fmt.Errorf("Error happens in WaitForLBToNotPending, should not reach here")
}

// CheckLBStatus check the loadbalancer status after a commmand execution.
func CheckLBStatus(cmdctx *CommandContext) {
	args := strings.Split(cmdctx.Command, " ")
	subcmd := ""
	for _, arg := range args {
		if strings.HasPrefix(arg, "lbaas-") {
			subcmd = arg
		}
	}
	subs := strings.Split(subcmd, "-")
	resourceType, operation := subs[1], subs[2]

	fs := time.Now()
	if operation == "create" || operation == "update" || operation == "delete" {
		if cmdctx.Check.LBIDName == "" {
			logger.Printf("Command(%d/%d): No loadbalancer appointed, no check to do.", cmdctx.Seq, len(cmdList))
			cmdctx.Check.Result = fmt.Sprintf("No loadbalancer appointed, no check to do.")
		} else if resourceType == "loadbalancer" && operation == "delete" {
			logger.Printf("Command(%d/%d): Loadbalancer deleted, no check to do.", cmdctx.Seq, len(cmdList))
			cmdctx.Check.Result = fmt.Sprintf("loadbalancer deleted, no check to do.")
		} else {
			checkCmd := fmt.Sprintf("neutron lbaas-loadbalancer-show %s", cmdctx.Check.LBIDName)
			logger.Printf("Command(%d/%d): Check with command: '%s'", cmdctx.Seq, len(cmdList), checkCmd)
			chkctx := CommandContext{
				Command: checkCmd,
			}
			maxTries := 32
			tried := 0
			for ; tried < maxTries; tried++ {
				RunCmd(&chkctx)
				if chkctx.ExitCode != 0 {
					logger.Printf("Command(%d/%d): Checked loadbalancer %s Failed: %s",
						cmdctx.Seq, len(cmdList), cmdctx.Check.LBIDName, chkctx.Err)
					cmdctx.Check.Result = fmt.Sprintf("Failed to check execution of %s: %s", cmdctx.Command, chkctx.Err)
					break
				}

				_ = json.Unmarshal([]byte(chkctx.RawOut), &cmdctx.Check.ShowResp)
				resp := cmdctx.Check.ShowResp
				logger.Printf("Command(%d/%d): Checked loadbalancer %s is %s",
					cmdctx.Seq, len(cmdList), cmdctx.Check.LBIDName, resp.ProvisioningStatus)
				if strings.HasPrefix(resp.ProvisioningStatus, "PENDING_") {
					time.Sleep(time.Duration(1) * time.Second)
					continue
				} else {
					cmdctx.Check.Result = fmt.Sprintf("LB: %s %s", resp.ID, resp.ProvisioningStatus)
					break
				}
			}
			if tried >= maxTries {
				cmdctx.Check.Result = fmt.Sprintf("LB: %s left PENDING", cmdctx.Check.LBIDName)
			}
		}
	} else { // 'show' 'list' no need to check
		cmdctx.Check.Result = fmt.Sprintf("%s done", subcmd)
	}
	fe := time.Now()

	cmdctx.Check.Duration = fe.Sub(fs) + cmdctx.CmdDuration
	logger.Printf("Command(%d/%d): Checked time: %d ms", cmdctx.Seq, len(cmdList), cmdctx.Check.Duration.Milliseconds())
}

// RunCmd run the command and fill CommandResult body
func RunCmd(cmdctx *CommandContext) {
	cmdArgs := strings.Split(cmdctx.Command, " ")
	cmdArgs = append(cmdArgs, "--format", "json")
	var out, err bytes.Buffer
	// c := exec.Command(cmdArgs[0], cmdArgs[1:]...)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(30)*time.Minute)
	defer cancel()
	c := exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)

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
		} else {
			cmdctx.RawOut = out.String()
		}
	}
	cmdctx.Err = err.String()
	fe := time.Now()
	cmdctx.ExitCode = c.ProcessState.ExitCode()
	cmdctx.CmdDuration = fe.Sub(fs)
}

// HandleArguments handle user's input.
func HandleArguments() {
	flag.StringVar(&output, "output-filepath", "/dev/stdout", "output the result")
	flag.IntVar(&maxCheckTimes, "max-check-times", maxCheckTimes, "The max times for checking loadbalancer is ready for next step.")
	flag.StringVar(&checkLB, "check-lb", "", "the loadbalancer name or id for checking execution status.")

	flag.Usage = PrintUsage
	flag.Parse()

	logger.Printf("output to: %s", output)

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
	logger.Printf("Command template: %s", neutronCmdArgs)

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

	logger.Printf("variables parsed as")
	for k, v := range variables {
		logger.Printf("%10s: %v", k, v)
	}

	ConstructFromTemplate(neutronCmdArgs, variables)
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
