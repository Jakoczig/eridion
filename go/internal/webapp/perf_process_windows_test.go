package webapp

import (
	"syscall"
	"unsafe"
)

type processMemoryCounters struct {
	cb                         uint32
	PageFaultCount             uint32
	PeakWorkingSetSize         uintptr
	WorkingSetSize             uintptr
	QuotaPeakPagedPoolUsage    uintptr
	QuotaPagedPoolUsage        uintptr
	QuotaPeakNonPagedPoolUsage uintptr
	QuotaNonPagedPoolUsage     uintptr
	PagefileUsage              uintptr
	PeakPagefileUsage          uintptr
}

var getProcessMemoryInfo = syscall.NewLazyDLL("psapi.dll").NewProc("GetProcessMemoryInfo")

// processSample returns cumulative CPU seconds (user+kernel), the current and
// the peak working set of this process.
func processSample() (cpuSeconds float64, workingSet, peakWorkingSet uint64) {
	h, err := syscall.GetCurrentProcess()
	if err != nil {
		return 0, 0, 0
	}
	var c, e, k, u syscall.Filetime
	if syscall.GetProcessTimes(h, &c, &e, &k, &u) == nil {
		cpuSeconds = float64(k.Nanoseconds()+u.Nanoseconds()) / 1e9
	}
	var m processMemoryCounters
	m.cb = uint32(unsafe.Sizeof(m))
	if r, _, _ := getProcessMemoryInfo.Call(uintptr(h), uintptr(unsafe.Pointer(&m)), uintptr(m.cb)); r != 0 {
		workingSet, peakWorkingSet = uint64(m.WorkingSetSize), uint64(m.PeakWorkingSetSize)
	}
	return
}
