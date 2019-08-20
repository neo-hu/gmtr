package gmtr

import (
	"github.com/neo-hu/gmtr/net/socket"
	"syscall"
)

func (m *GMtr) sysSelect(timeout *syscall.Timeval) (int, error) {
	rfds := &syscall.FdSet{}
	socket.FD_ZERO(rfds)
	socket.FD_SET(rfds, m.socketFd)
	nfound, err := syscall.Select(m.socketFd+1, rfds, nil, nil, timeout)
	if err != nil {
		return -1, err
	}
	if nfound > 0 {
		if socket.FD_ISSET(rfds, m.socketFd) {
			return m.socketFd, nil
		}
	}
	return -1, nil
}
