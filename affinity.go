// +build !linux

package empa

import "errors"

// ErrInvalidPID is returned when pinning to an unavailable CPU.
var ErrInvalidPID = errors.New("Invalid PID")

// pinToCPU does nothing on this architecture.
func pinToCPU(pid int, cpu uint) error {
	return nil
}
