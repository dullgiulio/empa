// +build !linux

package empa

import "errors"

var ErrInvalidPID = errors.New("Invalid PID")

// pinToCPU does nothing on this architecture.
func pinToCPU(pid int, cpu uint) error {
	return nil
}
