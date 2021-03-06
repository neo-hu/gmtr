package gmtr

import (
	"syscall"
	"time"
)

func WaitTime(waitTime time.Duration) *syscall.Timeval {
	timeout := &syscall.Timeval{}
	if waitTime > 0 {
		dur := int64(waitTime / time.Microsecond)
		timeout.Sec, timeout.Usec = dur/1000000, dur%1000000
	}
	return timeout
}
