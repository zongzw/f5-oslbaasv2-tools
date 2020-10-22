package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// StringArray []string
type StringArray []string

const fkDatetimeFormat = "2006-01-02 15:04:05"
const patternDatetimePID = `\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}.\d{3} \d+ `
const patternDatetime = `\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}.\d{3}`

var (
	regexpDatetimePID = regexp.MustCompile(patternDatetimePID)
	regexpDatetime    = regexp.MustCompile(patternDatetime)
)

func main() {
	var start, end string
	var single, gz bool
	var filters StringArray
	var logPaths StringArray
	var output string

	flag.BoolVar(&single, "single", false, "Saving to a single file?")
	flag.BoolVar(&gz, "gz", false, "gzip the output files?")
	flag.StringVar(&start, "begin-time", "", "start datetime, format: 2006-01-02 15:04:05.000")
	flag.StringVar(&end, "end-time", "", "end datetime, format: 2006-01-02 15:04:05.000")
	flag.Var(&filters, "filter", "filter keys, regexp supported.")
	flag.Var(&logPaths, "logpath", "log paths, regexp supported.")
	flag.StringVar(&output, "output-dirpath", ".", "output folder, will be created if not exists.")
	flag.Parse()

	if start == "" || end == "" {
		fmt.Printf("start time and end time cannot be empty.\n")
		return
	}
	fmt.Printf("start: %s, end: %s, single: %v, gz: %v, filters: %v, logpaths: %v, logdir: %s\n",
		start, end, single, gz, filters, logPaths, output)

	for _, lf := range logPaths {
		fh, e := os.Open(lf)
		if e != nil {
			fmt.Printf("Failed to open file for reading: %s\n", e.Error())
			continue
		}

		endT, _ := TimeStringToTime(end)
		beginT, _ := TimeStringToTime(start)
		ePos, err := SeekToDateTime(fh, endT)
		if err != nil {
			fmt.Printf("failed to seed to time: %s: %s\n", end, err.Error())
			continue
		}

		sPos, err := SeekToDateTime(fh, beginT)
		if err != nil {
			fmt.Printf("failed to seed to time: %s: %s\n", start, err.Error())
			continue
		}

		sStart, _, err := LineStartAndEnd(fh, sPos)
		if err != nil {
			fmt.Printf("File %s failed to determine the start line to copy.", lf)
			continue
		}
		_, eEnd, err := LineStartAndEnd(fh, ePos)
		if err != nil {
			fmt.Printf("File %s failed to determine the end line to copy.", lf)
			continue
		}

		fw, err := os.OpenFile(
			strings.Join([]string{output, filepath.Base(lf)}, "/"),
			os.O_CREATE|os.O_WRONLY, os.ModePerm)

		err = TailAt(fh, fw, sStart, eEnd)
		if err != nil {
			fmt.Printf("File %s reading from %d to %d failed: %s\n", lf, sStart, eEnd, err.Error())
		}
	}

}

// TailAt tail the file content from start to end to fw.
func TailAt(fr, fw *os.File, start, end int64) error {
	size := 512 * 1024
	buff := make([]byte, size)
	stat, _ := fr.Stat()

	for start < end {
		n, err := fr.ReadAt(buff, start)
		if err == io.EOF {
		} else if err != nil {
			return fmt.Errorf("file %s reading from %d to %d failed: %s",
				stat.Name(), start, end, err.Error())
		}
		if start+int64(n) < end {
			fw.WriteString(string(buff[0:n]))
		} else {
			fw.WriteString(string(buff[0 : end-start]))
		}
		start = start + int64(n)
	}
	return nil
}

// SeekToDateTime get the position with the given datetime.
// The position may not the beginning of the line.
func SeekToDateTime(fh *os.File, datetime time.Time) (int64, error) {

	stat, _ := fh.Stat()
	filesize := stat.Size()

	sPos := int64(0)
	ePos := filesize - 1
	cPos := (sPos + ePos) / 2

	retryMax := 64 // incase the whole file has no timestamp at all.
	retry := 0
	for sPos < ePos && retry < retryMax {
		sameLine, err := IsInSameLine(fh, sPos, ePos)
		if err != nil {
			return -1, err
		}
		if sameLine {
			return cPos, nil
		}
		cPos = (sPos + ePos) / 2
		var dt time.Time
		for cPos > sPos {
			dt, err = TimeOfPosition(fh, cPos)
			if err != nil {
				retry++
				cPos, err = PreviousLinePos(fh, cPos)
				if err != nil {
					return -1, err
				}
			} else {
				retry = 0
				break
			}
		}

		if dt.Before(datetime) {
			sPos = cPos + 1
		} else if dt.After(datetime) {
			ePos = cPos - 1
		} else {
			return cPos, nil
		}
	}

	if retry >= retryMax {
		fmt.Printf(
			"File %s max retries(%d) while finding timestamp, "+
				"but no found, check the log has timestamps",
			stat.Name(), retryMax)
		return cPos, nil
	}

	return cPos, nil
}

// IsInSameLine check  the mPos and nPos are in the same line.
func IsInSameLine(fh *os.File, mPos, nPos int64) (bool, error) {
	a, _, e1 := LineStartAndEnd(fh, mPos)
	if e1 != nil {
		return false, e1
	}
	b, _, e2 := LineStartAndEnd(fh, nPos)
	if e2 != nil {
		return false, e2
	}

	if a == b {
		return true, nil
	} else {
		return false, nil
	}
}

// PreviousLinePos return the previous line pos.
func PreviousLinePos(fh *os.File, nPos int64) (int64, error) {
	a, _, err := LineStartAndEnd(fh, nPos)
	return a - 1, err
}

// TimeOfPosition return the timestamp parsed from the line around nPos
func TimeOfPosition(fh *os.File, nPos int64) (time.Time, error) {
	a, b, err := LineStartAndEnd(fh, nPos)
	if err != nil {
		return time.Now(), err
	}

	buff := make([]byte, b-a)

	fh.Seek(a, 0)
	fh.ReadAt(buff, a)

	return DateTimeOfLine(string(buff))
}

// LineStartAndEnd return the line positions(start and end) where nPos locates
func LineStartAndEnd(fh *os.File, nPos int64) (int64, int64, error) {
	stat, _ := fh.Stat()
	if nPos < 0 || nPos >= stat.Size() {
		return -1, -1, fmt.Errorf(
			"File %s invalid file position: %d size: %d",
			stat.Name(), nPos, stat.Size())
	}

	var start, end int64
	buffSize := int64(128)
	buff := make([]byte, buffSize)

	start = nPos
	for start > 0 {
		if start < buffSize {
			start = 0
		} else {
			start = start - buffSize
		}

		_, err := fh.ReadAt(buff, start)
		if err != nil {
			return -1, -1, fmt.Errorf(
				"File %s ReadAt %d failed: %s", stat.Name(), start, err.Error())
		}

		i := strings.LastIndex(string(buff), "\n")
		if i != -1 {
			start = start + int64(i+1)
			break
		}
	}
	end = nPos
	for end < stat.Size() {
		n, err := fh.ReadAt(buff, end)
		if err != nil && err != io.EOF {
			return -1, -1, fmt.Errorf(
				"File %s ReadAt %d failed: %s", stat.Name(), end, err.Error())
		}
		i := strings.Index(string(buff), "\n")
		if i != -1 {
			end = end + int64(i)
			break
		} else {
			end = end + int64(n)
		}
	}

	return start, end, nil
}

// DateTimeOfLine return timestamp of the log.
func DateTimeOfLine(log string) (time.Time, error) {
	var (
		dt  time.Time = time.Now()
		err error     = nil
	)

	dts := regexpDatetimePID.FindString(log)
	if dts == "" {
		l := len(log)
		lp := 50
		if lp >= l {
			lp = l
		}
		dt, err = time.Now(), fmt.Errorf("No datetime found in the string: %s", log[0:lp])
	} else {
		dts = regexpDatetime.FindString(dts)
		dt, err = time.Parse(fkDatetimeFormat, dts)
		if err != nil {
			dt, err = time.Now(), fmt.Errorf("invalid time string: %s", dts)
		}
	}

	return dt, err
}

// TimeStringToTime convert time string to time object.
func TimeStringToTime(dts string) (time.Time, error) {
	return time.Parse(fkDatetimeFormat, dts)
}

// Set StringArray's Set interface for flag.
func (sa *StringArray) Set(s string) error {
	*sa = append(*sa, s)
	return nil
}

func (sa *StringArray) String() string {
	return fmt.Sprintf("%s", *sa)
}
