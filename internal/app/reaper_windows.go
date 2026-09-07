//go:build windows

package app

import (
	"log"
	"unsafe"

	"golang.org/x/sys/windows"
)

// startDeathReaper puts dsh in a job with KILL_ON_JOB_CLOSE. Force-killing
// the shell closes the job handle and Windows tears down the process tree.
func startDeathReaper(pid int) func() {
	if pid <= 0 {
		return func() {}
	}
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		log.Printf("dsh death reaper: job: %v", err)
		return func() {}
	}
	var info windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION
	info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if _, err := windows.SetInformationJobObject(
		job,
		windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info)),
	); err != nil {
		_ = windows.CloseHandle(job)
		log.Printf("dsh death reaper: job limits: %v", err)
		return func() {}
	}
	proc, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(pid))
	if err != nil {
		_ = windows.CloseHandle(job)
		log.Printf("dsh death reaper: open process: %v", err)
		return func() {}
	}
	err = windows.AssignProcessToJobObject(job, proc)
	_ = windows.CloseHandle(proc)
	if err != nil {
		_ = windows.CloseHandle(job)
		log.Printf("dsh death reaper: assign: %v", err)
		return func() {}
	}
	return func() {
		_ = windows.CloseHandle(job)
	}
}
