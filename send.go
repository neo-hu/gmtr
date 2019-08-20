package gmtr

import (
	ipv42 "github.com/neo-hu/gmtr/net/ipv4"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
	"syscall"
)

func (m *GMtr) sendV6() error {
	seq := m.incr()
	sv := &seqValue{
		ttl:    m.currentTTL,
		ping:   m.currentCount,
		time:   m.lastSendTime,
		evTime: m.lastSendTime.Add(m.timeout),
	}
	m.seqMap.Store(seq, sv)
	m.evEnqueue(sv)
	m.setSendResult(m.currentTTL)

	b, err := (&icmp.Message{
		Type: ipv6.ICMPTypeEchoRequest, Code: 0,
		Body: &icmp.Echo{
			ID: m.ident, Seq: seq,
			Data: make([]byte, m.dataSize),
		},
	}).Marshal(nil)
	if err != nil {
		return err
	}
	err = syscall.SetsockoptInt(m.socketFd, syscall.IPPROTO_IPV6, syscall.IPV6_UNICAST_HOPS, m.currentTTL)
	if err != nil {
		return err
	}
	return syscall.Sendto(m.socketFd, b, 0, m.sa)
}

func (m *GMtr) sendV4() error {
	seq := m.incr()
	sv := &seqValue{
		ttl:    m.currentTTL,
		ping:   m.currentCount,
		time:   m.lastSendTime,
		evTime: m.lastSendTime.Add(m.timeout),
	}
	m.seqMap.Store(seq, sv)
	m.evEnqueue(sv)
	m.setSendResult(m.currentTTL)
	b, err := (&icmp.Message{
		Type: ipv4.ICMPTypeEcho, Code: 0,
		Body: &icmp.Echo{
			ID: m.ident, Seq: seq,
			Data: make([]byte, m.dataSize),
		},
	}).Marshal(nil)
	if err != nil {
		return err
	}
	header := &ipv4.Header{
		Version:  ipv4.Version,
		Len:      ipv4.HeaderLen,
		Protocol: syscall.IPPROTO_ICMP,
		TotalLen: ipv4.HeaderLen + len(b),
		ID:       seq + 3232,
		TTL:      m.currentTTL,
		Dst:      m.ip.To4(),
	}
	return ipv42.WriteTo(m.socketFd, header, b, nil)
}

func (m *GMtr) incr() int {
	m.Lock()
	defer m.Unlock()
	m.cnt += 1
	return m.cnt
}
