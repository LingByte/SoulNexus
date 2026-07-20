package system

import (
	"bytes"
	"runtime"
	"strings"
)

func buildGoroutineSnapshot(current int64) GoroutineSnapshot {
	states := map[string]int{}
	buf := make([]byte, 512*1024)
	n := runtime.Stack(buf, true)
	lines := bytes.Split(buf[:n], []byte("\n"))
	for _, line := range lines {
		s := string(line)
		if !strings.HasPrefix(s, "goroutine ") {
			continue
		}
		lb := strings.Index(s, "[")
		rb := strings.Index(s, "]")
		if lb < 0 || rb <= lb {
			states["unknown"]++
			continue
		}
		state := strings.TrimSpace(s[lb+1 : rb])
		if state == "" {
			state = "unknown"
		}
		states[state]++
	}

	leakSuspect := current > 50 && int64(goroutineHighWater) >= current &&
		float64(current) >= float64(goroutineHighWater)*0.95

	gr := GoroutineSnapshot{
		NumGoroutine:    int(current),
		NumGoroutineMax: int(goroutineHighWater),
		NumCgoCall:      runtime.NumCgoCall(),
		ByState:         states,
		LeakSuspect:     leakSuspect,
	}
	if p, err := processSelfThreads(); err == nil {
		gr.NumThread = int(p)
	} else {
		gr.NumThread = runtime.GOMAXPROCS(0)
	}
	return gr
}
