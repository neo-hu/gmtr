package ipv4

import (
	"github.com/neo-hu/gmtr/net/socket"
	"golang.org/x/net/ipv4"
	"net"
)

func WriteTo(fd int, h *ipv4.Header, p []byte, cm *ControlMessage) error {
	m := socket.Message{
		OOB: cm.Marshal(),
	}
	wh, err := h.Marshal()
	if err != nil {
		panic(err)
	}
	dst := new(net.IPAddr)
	if cm != nil {
		if ip := cm.Dst.To4(); ip != nil {
			dst.IP = ip
		}
	}
	if dst.IP == nil {
		dst.IP = h.Dst
	}
	m.Addr = dst
	m.Buffers = [][]byte{wh, p}
	return socket.SendMsg(fd, &m, 0)
}
