// +build linux

package empa

import (
	"errors"
	"syscall"
	"unsafe"
)

// ErrInvalidPID is returned when pinning to an unavailable CPU.
var ErrInvalidPID = errors.New("Invalid PID")

// pinToCPU sets the affinity of pid to be locked to cpu.
func pinToCPU(pid int, cpu uint) error {
	var mask [1024 / 64]uintptr
	if pid <= 0 {
		return ErrInvalidPID
	}
	mask[cpu/64] |= 1 << (cpu % 64)
	_, _, err := syscall.RawSyscall(syscall.SYS_SCHED_SETAFFINITY,
		uintptr(pid),
		uintptr(len(mask)*8),
		uintptr(unsafe.Pointer(&mask[0])))
	if err != 0 {
		return err
	}
	return nil
}
