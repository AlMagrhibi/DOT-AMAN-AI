// Package metrics reads local process counters; it never sends telemetry.
package metrics

import (
	"runtime"
	"syscall"
	"time"
)

type Snapshot struct {
	cpu     time.Duration
	started time.Time
}
type Process struct {
	CPUPercent   float64 `json:"app_cpu_percent_one_core"`
	PeakRSSBytes int64   `json:"app_lifetime_peak_rss_bytes"`
	GoHeapBytes  uint64  `json:"app_go_heap_bytes"`
	Available    bool    `json:"available"`
}

func cpu(u syscall.Rusage) time.Duration {
	return time.Duration(u.Utime.Sec+u.Stime.Sec)*time.Second + time.Duration(u.Utime.Usec+u.Stime.Usec)*time.Microsecond
}
func Start() Snapshot {
	var u syscall.Rusage
	_ = syscall.Getrusage(syscall.RUSAGE_SELF, &u)
	return Snapshot{cpu(u), time.Now()}
}
func (s Snapshot) Finish() Process {
	var u syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &u); err != nil {
		return Process{}
	}
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	elapsed := time.Since(s.started)
	percent := 0.0
	if elapsed > 0 {
		percent = float64(cpu(u)-s.cpu) / float64(elapsed) * 100
	}
	return Process{percent, u.Maxrss * 1024, mem.Alloc, true}
}
