package socket

import (
	"syscall"
)

func fdget(fd int, fds *syscall.FdSet) (index, offset int) {
	index = fd / (syscall.FD_SETSIZE / len(fds.Bits)) % len(fds.Bits)
	offset = fd % (syscall.FD_SETSIZE / len(fds.Bits))
	return
}

func FD_SET(p *syscall.FdSet, i int) {
	idx, pos := fdget(i, p)
	p.Bits[idx] = 1 << uint(pos)
}

func FD_ISSET(p *syscall.FdSet, i int) bool {
	idx, pos := fdget(i, p)
	return p.Bits[idx]&(1<<uint(pos)) != 0
}

func FD_ZERO(p *syscall.FdSet) {
	for i := range p.Bits {
		p.Bits[i] = 0
	}
}
