package main

import (
	"flag"
	"fmt"
	"io"
	"log"
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
	logger            = log.New(os.Stdout, "", log.LstdFlags)
)

func main() {

	var begindts, enddts string
	var gz bool
	var filters StringArray
	var logPaths StringArray
	var output string

	ndt := time.Now()
	defaultdts := ndt.Format(fkDatetimeFormat)
	flag.BoolVar(&gz, "gz", false, "gzip the output files?")
	flag.StringVar(&begindts, "begin-time", "2000-01-01 00:00:00.000", "start datetime, format: 2006-01-02 15:04:05[.000].")
	flag.StringVar(&enddts, "end-time", defaultdts, "end datetime, format: 2006-01-02 15:04:05[.000]")
	flag.Var(&filters, "filter", "filter keys, regexp supported.")
	flag.Var(&logPaths, "logpath", "log paths, regexp supported.")
	flag.StringVar(&output, "output-dirpath", ".", "output folder, will be created if not exists.")
	flag.Parse()

	logger.Printf("start: %s, end: %s, gz: %v, filters: %v, logpaths: %v, logdir: %s\n",
		begindts, enddts, gz, filters, logPaths, output)

	for _, lf := range logPaths {
		fh, e := os.Open(lf)
		if e != nil {
			logger.Printf("Failed to open file for reading: %s\n", e.Error())
			continue
		}
		defer fh.Close()

		enddt, _ := TimeStringToTime(enddts)
		begindt, _ := TimeStringToTime(begindts)

		tsSeek := time.Now()
		ePos, err := SeekToDateTime(fh, enddt)
		if err != nil {
			logger.Printf("failed to seed to time: %s: %s\n", enddts, err.Error())
			continue
		} else {
			logger.Printf("Seeking %s took %d millisec.\n", enddts, time.Now().Sub(tsSeek).Milliseconds())
		}

		tsSeek = time.Now()
		sPos, err := SeekToDateTime(fh, begindt)
		if err != nil {
			logger.Printf("failed to seed to time: %s: %s\n", begindts, err.Error())
			continue
		} else {
			logger.Printf("Seeking %s took %d millisec.\n", begindts, time.Now().Sub(tsSeek).Milliseconds())
		}

		sStart, _, err := LineStartAndEnd(fh, sPos)
		if err != nil {
			logger.Printf("File %s failed to determine the start line to copy.", lf)
			continue
		}
		_, eEnd, err := LineStartAndEnd(fh, ePos)
		if err != nil {
			logger.Printf("File %s failed to determine the end line to copy.", lf)
			continue
		}

		begindtsRefmt := fmt.Sprintf("%d%d%d%d%d%d",
			begindt.Year(), begindt.Month(), begindt.Day(), begindt.Hour(), begindt.Minute(), begindt.Second())
		enddtsRefmt := fmt.Sprintf("%d%d%d%d%d%d",
			enddt.Year(), enddt.Month(), enddt.Day(), enddt.Hour(), enddt.Minute(), enddt.Second())
		outFileName := filepath.Base(lf) + "-" + begindtsRefmt + "-" + enddtsRefmt
		fw, err := os.OpenFile(
			strings.Join([]string{output, outFileName}, "/"),
			os.O_CREATE|os.O_WRONLY, os.ModePerm)
		defer fw.Close()

		err = TailAt(fh, fw, sStart, eEnd)
		if err != nil {
			logger.Printf("File %s reading from %d to %d failed: %s\n", lf, sStart, eEnd, err.Error())
		} else {
			logger.Printf("File %s reading from %d to %d, bytes: %d\n", lf, sStart, eEnd, eEnd-sStart)
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

	for sPos < ePos {
		sameLine, err := IsInSameLine(fh, sPos, ePos)
		if err != nil {
			return -1, err
		}
		if sameLine {
			return cPos, nil
		}
		cPos = (sPos + ePos) / 2
		// fmt.Println(sPos, cPos, ePos, ePos-sPos)
		var dt time.Time
		dt, err = TimeOfPosition(fh, cPos)
		if err != nil {
			pdt, pPos, err := PreviousDateTimeLine(fh, cPos)
			if err != nil {
				return -1, fmt.Errorf("file %s, failed to get previous time line: %d",
					fh.Name(), cPos)
			}
			ndt, nPos, err := NextDateTimeLine(fh, cPos)
			if err != nil {
				return -1, fmt.Errorf("file %s, failed to get next time line: %d",
					fh.Name(), cPos)
			}
			if pPos == -1 {
				return 0, nil
			} else if nPos >= stat.Size() {
				return stat.Size() - 1, nil
			} else {
				if pdt.After(datetime) {
					ePos = pPos - 1
					continue
				} else if ndt.Before(datetime) {
					sPos = nPos + 1
					continue
				} else {
					return cPos, nil
				}
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

	return cPos, nil
}

// PreviousDateTimeLine return the previous line which contains datetime.
func PreviousDateTimeLine(fh *os.File, nPos int64) (time.Time, int64, error) {

	prevPos := nPos
	for prevPos >= 0 {
		dt, err := TimeOfPosition(fh, prevPos)
		if err != nil {
			prevPos, err = PreviousLinePos(fh, prevPos)
			if err != nil {
				return time.Now(), prevPos, err
			} else if prevPos == -1 {
				return time.Now(), prevPos, nil
			}
		} else {
			return dt, prevPos, nil
		}
	}
	return time.Now(), -1, fmt.Errorf("file %s, Failed to find the previous time line, nPos: %d", fh.Name(), nPos)
}

// NextDateTimeLine return the next line which contains datetime.
func NextDateTimeLine(fh *os.File, nPos int64) (time.Time, int64, error) {
	nextPos := nPos
	stat, _ := fh.Stat()
	for nextPos < stat.Size() {
		dt, err := TimeOfPosition(fh, nextPos)
		if err != nil {
			nextPos, err = NextLinePos(fh, nextPos)
			if err != nil {
				return time.Now(), nextPos, err
			} else if nextPos >= stat.Size() {
				return time.Now(), stat.Size(), nil
			}
		} else {
			return dt, nextPos, nil
		}
	}
	return time.Now(), -1, fmt.Errorf("file %s, Failed to find the next time line, nPos: %d", fh.Name(), nPos)
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

	return a == b, nil
}

// PreviousLinePos return the previous line pos.
func PreviousLinePos(fh *os.File, nPos int64) (int64, error) {
	a, _, err := LineStartAndEnd(fh, nPos)
	return a - 1, err
}

// NextLinePos return a position belong to next line.
func NextLinePos(fh *os.File, nPos int64) (int64, error) {
	_, b, err := LineStartAndEnd(fh, nPos)
	return b + 1, err
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
	nPosInBuff := int64(0)
	for start > 0 {
		if start < buffSize {
			nPosInBuff = start
			start = 0
		} else {
			start = start - buffSize
			nPosInBuff = buffSize
		}

		_, err := fh.ReadAt(buff, start)
		if err != nil {
			return -1, -1, fmt.Errorf(
				"File %s ReadAt %d failed: %s", stat.Name(), start, err.Error())
		}

		i := strings.LastIndex(string(buff[0:nPosInBuff]), "\n")
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
