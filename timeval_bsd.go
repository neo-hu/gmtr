// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build darwin

package gmtr

import (
	"syscall"
	"time"
)

func WaitTime(waitTime time.Duration) *syscall.Timeval {
	timeout := &syscall.Timeval{}
	if waitTime > 0 {
		dur := int64(waitTime / time.Microsecond)
		timeout.Sec = dur / 1000000
		timeout.Usec = int32(dur % 1000000)
	}
	return timeout
}
