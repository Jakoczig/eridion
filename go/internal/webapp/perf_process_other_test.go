//go:build !windows

package webapp

import "syscall"

func processSample() (cpuSeconds float64, workingSet, peakWorkingSet uint64) {
	var r syscall.Rusage
	if syscall.Getrusage(syscall.RUSAGE_SELF, &r) == nil {
		cpuSeconds = float64(r.Utime.Sec+r.Stime.Sec) + float64(r.Utime.Usec+r.Stime.Usec)/1e6
		peakWorkingSet = uint64(r.Maxrss) * 1024
	}
	return
}
