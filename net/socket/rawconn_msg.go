package socket

import (
	"os"
)

func SendMsg(fd int, m *Message, flags int) error {
	var h msghdr
	vs := make([]iovec, len(m.Buffers))
	var sa []byte
	if m.Addr != nil {
		sa = marshalInetAddr(m.Addr)
	}
	h.pack(vs, m.Buffers, m.OOB, sa)
	n, operr := sendmsg(uintptr(fd), &h, flags)

	if operr != nil {
		return os.NewSyscallError("sendmsg", operr)
	}
	m.N = n
	m.NN = len(m.OOB)
	return nil
}
