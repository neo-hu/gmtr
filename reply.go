package gmtr

import (
	"encoding/binary"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
	"golang.org/x/sys/unix"
	"net"
	"syscall"
	"time"
)

func (m *GMtr) socketCanRead(waitTime time.Duration) int {
	timeout := WaitTime(waitTime)
selectAgain:
	fd, err := m.sysSelect(timeout)
	if err != nil && err == unix.EINTR {
		goto selectAgain
	}
	return fd
}

func (m *GMtr) waitForReply(waitTime time.Duration) bool {
	s := m.socketCanRead(waitTime)
	if s <= 0 {
		// todo timeout
		return false
	}
	m.procRecv(s)
	return s > 0
}

func (m *GMtr) setSendResult(ttl int) {
	if m.result[ttl] == nil {
		m.result[ttl] = &seqResult{ips: make(StringSet)}
	}
	m.result[ttl].count += 1
}

func (m *GMtr) setResult(ttl int, ip net.IP, reply time.Duration) {
	item := m.result[ttl]
	if item == nil {
		return
	}
	if item.min == 0 || item.min > reply {
		item.min = reply
	}
	if item.max == 0 || item.max < reply {
		item.max = reply
	}
	item.reply += 1
	item.sum += reply
	item.ips.Add(ip.String())
}

func (m *GMtr) procRecv(fd int) error {
	buffer := make([]byte, 3000)
	n, ra, err := syscall.Recvfrom(fd, buffer, 0)
	if err != nil {
		time.Sleep(m.interval)
		return err
	}
	switch ra.(type) {
	case *syscall.SockaddrInet6:
		m1, err := icmp.ParseMessage(ipv6.ICMPType(0).Protocol(), buffer)
		if err != nil {
			return err
		}
		switch m1.Type {
		case ipv6.ICMPTypeEchoReply:
			if pkt, ok := m1.Body.(*icmp.Echo); ok {
				if pkt.ID != m.ident {
					// todo 非本程序发送的
					return nil
				}
				if val, ok := m.seqMap.Load(int(pkt.Seq)); ok {
					// todo 已经到目的了
					m.seqMap.Delete(int(pkt.Seq))
					sv := val.(*seqValue)
					m.lastTTL[sv.ping] = struct{}{}
					m.evRemove(sv)
					m.setResult(sv.ttl, net.IP(ra.(*syscall.SockaddrInet6).Addr[:]), time.Now().Sub(sv.time))
					if m.targetTTL > sv.ttl {
						m.targetTTL = sv.ttl
					}
				}
			}
			break
		case ipv6.ICMPTypeTimeExceeded:
			if len(buffer) < 55 {
				// todo 未知
				return nil
			}
			if int(binary.BigEndian.Uint16(buffer[52:54])) != m.ident {
				// todo 非本程序发送的
				return nil
			}
			pkgSeq := binary.BigEndian.Uint16(buffer[54:56])
			if val, ok := m.seqMap.Load(int(pkgSeq)); ok {
				sv := val.(*seqValue)
				m.evRemove(sv)
				m.seqMap.Delete(int(pkgSeq))
				m.setResult(sv.ttl, net.IP(ra.(*syscall.SockaddrInet6).Addr[:]), time.Now().Sub(sv.time))
			}
			break
		}
	case *syscall.SockaddrInet4:
		src, _ := stripIPv4Header(n, buffer)
		m1, err := icmp.ParseMessage(ipv4.ICMPType(0).Protocol(), buffer)
		if err != nil {
			return err
		}
		switch m1.Type {
		case ipv4.ICMPTypeTimeExceeded:
			// todo ttl 超时
			if len(buffer) < 36 {
				// todo 未知
				return nil
			}
			if int(binary.BigEndian.Uint16(buffer[32:34])) != m.ident {
				// todo 非本程序发送的
				return nil
			}
			pkgSeq := binary.BigEndian.Uint16(buffer[34:36])
			if val, ok := m.seqMap.Load(int(pkgSeq)); ok {
				sv := val.(*seqValue)
				m.evRemove(sv)
				m.seqMap.Delete(int(pkgSeq))

				m.setResult(sv.ttl, src, time.Now().Sub(sv.time))
			}
			break
		case ipv4.ICMPTypeEchoReply:
			if pkt, ok := m1.Body.(*icmp.Echo); ok {
				if pkt.ID != m.ident {
					// todo 非本程序发送的
					return nil
				}
				if val, ok := m.seqMap.Load(int(pkt.Seq)); ok {
					// todo 已经到目的了
					//m.currentTTL = 1
					//m.currentPingCount += 1
					m.seqMap.Delete(int(pkt.Seq))
					sv := val.(*seqValue)
					m.lastTTL[sv.ping] = struct{}{}
					m.evRemove(sv)
					m.setResult(sv.ttl, src, time.Now().Sub(sv.time))
					if m.targetTTL > sv.ttl {
						m.targetTTL = sv.ttl
					}
				}
			}
		}
	}
	return nil
}

func stripIPv4Header(n int, b []byte) (net.IP, int) {
	if len(b) < 20 {
		return nil, n
	}
	l := int(b[0]&0x0f) << 2
	if 20 > l || l > len(b) {
		return nil, n
	}
	if b[0]>>4 != 4 {
		return nil, n
	}
	src := net.IPv4(b[12], b[13], b[14], b[15])
	copy(b, b[l:])
	return src, n - l
}
