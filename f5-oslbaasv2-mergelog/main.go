package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"time"
)

type arrayFlags []string

var (
	logpaths           = arrayFlags{}
	outpath            = "./merged.log"
	maxCapacity        = 512 * 1024
	patternDatetimePID = `\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}.\d{3} \d+ `
	patternDatetime    = `\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}.\d{3}`
	validReg           = regexp.MustCompile(patternDatetimePID)
	validDT            = regexp.MustCompile(patternDatetime)
	datetimeLayout     = "2006-01-02 15:04:05"
)

// LogEntryContext ...
type LogEntryContext struct {
	inited          bool
	handler         *os.File
	filename        string
	EOF             bool
	buff            []byte
	ready           bool
	scanner         *bufio.Scanner
	notimes         []string
	lastErr         error
	beginTime       time.Time
	beginTimeSetted bool
}

func main() {
	fmt.Printf("hello world.\n")
	flag.Var(&logpaths, "logpath", "logpaths to merge")
	flag.StringVar(&outpath, "output-filepath", outpath, "the output merged file path")

	flag.Parse()

	log.Printf("%s, %s", logpaths, outpath)

	lcs := []*LogEntryContext{}
	for _, fn := range logpaths {
		lc, err := NewLogEntryContext(fn)
		if err != nil {
			log.Printf("Failed to merge %s: %s", fn, err.Error())
			continue
		}
		lc.Next()
		lcs = append(lcs, lc)
		defer lc.handler.Close()
	}

	lines := 0
	out, err := os.OpenFile(outpath, os.O_CREATE|os.O_WRONLY, os.ModePerm)
	if err != nil {
		log.Fatalf("Failed to open file %s for writing: %s", outpath, err.Error())
	}
	defer out.Close()

	for true {
		li := leastLog(lcs)
		if li == -1 {
			break
		}
		// log.Print(lcs[li].scanner.Text())
		rt, _ := lcs[li].RelTime()
		// at, _ := lcs[li].AbsTime()
		// log.Printf("file %d: reltime: %v, abstime: %v", li, rt, at)
		out.WriteString(fmt.Sprintf("<<%d>> %07d: %s\n", li, rt.Milliseconds(), lcs[li].scanner.Text()))
		lines = lines + 1
		lcs[li].Next()
	}
	log.Print(lines)
}

// NewLogEntryContext ..
func NewLogEntryContext(filename string) (*LogEntryContext, error) {

	handler, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	l := LogEntryContext{
		inited:          true,
		filename:        filename,
		buff:            make([]byte, maxCapacity),
		notimes:         []string{},
		scanner:         bufio.NewScanner(handler),
		EOF:             false,
		ready:           false,
		beginTimeSetted: false,
		handler:         handler,
	}

	return &l, nil
}

// Next ...
func (lc *LogEntryContext) Next() {
	if !lc.inited || lc.EOF {
		return
	}

	lc.ready = lc.scanner.Scan()
	if !lc.ready {
		if err := lc.scanner.Err(); err != nil {
			lc.inited = false
		} else {
			lc.EOF = true
		}
	} else {
		lc.notimes = []string{}
		for !lc.EOF && validReg.Find(lc.scanner.Bytes()) == nil {
			lc.notimes = append(lc.notimes, lc.scanner.Text())
			if !lc.scanner.Scan() {
				if err := lc.scanner.Err(); err != nil {
					lc.ready = false
					lc.lastErr = err
					break
				} else {
					lc.EOF = true
				}
			}
		}
		if !lc.beginTimeSetted {
			dtb := validDT.Find(lc.scanner.Bytes())
			if dtb == nil {
				lc.lastErr = fmt.Errorf("Should not happen here, no valid time string found")
			} else {
				lc.beginTime, _ = time.Parse(datetimeLayout, string(dtb))
				lc.beginTimeSetted = true
			}
		}
	}
}

// AbsTime ...
func (lc *LogEntryContext) AbsTime() (*time.Time, error) {
	if lc.EOF {
		return nil, fmt.Errorf("End Of file")
	}
	dtb := validDT.Find(lc.scanner.Bytes())
	dt, err := time.Parse(datetimeLayout, string(dtb))
	return &dt, err
}

// RelTime ...
func (lc *LogEntryContext) RelTime() (*time.Duration, error) {
	dt, err := lc.AbsTime()
	if err != nil {
		return nil, err
	}

	dur := dt.Sub(lc.beginTime)
	return &dur, nil
}

func (i *arrayFlags) String() string {
	return fmt.Sprint(*i)
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

// func test() {
// 	l, e := NewLogEntryContext("./tests/x.log")
// 	if e != nil {
// 		log.Printf("error in tests: %s", e.Error())
// 	}
// 	for !l.EOF {
// 		l.Next()
// 		// for _, n := range l.notimes {
// 		// 	log.Print(n)
// 		// }
// 		log.Print(l.scanner.Text())
// 		log.Print(l.Time())
// 	}
// }

func leastLog(lcs []*LogEntryContext) int {
	if len(lcs) == 0 {
		return -1
	}

	min := time.Duration(10*12*30*24*3600) * time.Second
	li := -1
	for i, v := range lcs {
		if v.EOF {
			continue
		}
		vTime, _ := v.RelTime()
		if min > *vTime {
			min = *vTime
			li = i
		}
	}

	return li
}
