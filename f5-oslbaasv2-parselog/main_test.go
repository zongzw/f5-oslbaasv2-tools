package main

import (
	"testing"
	"time"
)

func Test_FKTheTime(t *testing.T) {
	dts := map[string]time.Time{
		"2022-09-29 16:28:33.492": time.Date(2022, 9, 29, 16, 28, 33, int(492*time.Millisecond), time.UTC),
		"":                        time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	for st, tt := range dts {

		fed := FKTheTime(st)
		t.Logf("%s -> %v", st, fed)
		if !fed.Equal(tt) {
			t.FailNow()
		}
	}
}
