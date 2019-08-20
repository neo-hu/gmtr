package gmtr

import (
	"fmt"
	ipv42 "github.com/neo-hu/gmtr/net/ipv4"
	"github.com/pkg/errors"
	"golang.org/x/sys/unix"
	"net"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"
)

type Mode int

const (
	IPV4Mode Mode = 4
	IPV6Mode Mode = 6
)

type GMtr struct {
	sync.RWMutex
	interval     time.Duration //发送icmp包的间隔
	timeout      time.Duration //超时时间
	ident        int           // ping标记
	socketFd     int           // ipv4 socket
	forceIPv4    bool          // 是否强制使用ipv4地址
	forceIPv6    bool          // 是否强制使用ipv6地址
	maxTTL       int           // 最大ttl
	currentTTL   int           // 当前发送ttl
	targetTTL    int           // 对端的ttl
	dataSize     int           // 发送数据的大小
	count        int           // 每个ttl发包的数量
	currentCount int           // 当前的count
	mode         Mode          // ipv4 还是ipv6
	ip           net.IP
	localIp      net.IP
	sa           syscall.Sockaddr

	lastTTL map[int]struct{} // 如果已经到达对端，当前ping循环不用不用继续发送ttl

	seqMap       sync.Map
	lastSendTime time.Time
	result       []*seqResult // todo 结果，[ttl]*seqResult

	cnt int // seq id

	evFirst *seqValue
	evLast  *seqValue
}

type Option interface {
	apply(opt *GMtr)
}
type optionFunc func(*GMtr)

func (f optionFunc) apply(opt *GMtr) {
	f(opt)
}

func ForceIPv4Option(f bool) Option {
	return optionFunc(func(m *GMtr) {
		m.forceIPv4 = f
	})
}

func ForceIPv6Option(f bool) Option {
	return optionFunc(func(m *GMtr) {
		m.forceIPv6 = f
	})
}

func DataSizeOption(s int) Option {
	return optionFunc(func(m *GMtr) {
		m.dataSize = s
	})
}
func CountOption(c int) Option {
	return optionFunc(func(m *GMtr) {
		m.count = c
	})
}

func IntervalOption(c time.Duration) Option {
	return optionFunc(func(m *GMtr) {
		m.interval = c
	})
}
func TimeoutOption(c time.Duration) Option {
	return optionFunc(func(m *GMtr) {
		m.timeout = c
	})
}
func MaxTTLOption(c int) Option {
	return optionFunc(func(m *GMtr) {
		m.maxTTL = c
	})
}
func IdentOption(c int) Option {
	return optionFunc(func(m *GMtr) {
		m.ident = c
	})
}

func IsIPv4(ip net.IP) bool {
	return len(ip.To4()) == net.IPv4len
}

// IsIPv6 returns true if ip version is v6
func IsIPv6(ip net.IP) bool {
	if r := strings.Index(ip.String(), ":"); r != -1 {
		return true
	}
	return false
}

func NewGMtr(target string, opts ...Option) (*GMtr, error) {
	m := &GMtr{
		ident:        os.Getpid() & 0xFFFF,
		currentTTL:   1,
		maxTTL:       30,
		lastTTL:      map[int]struct{}{},
		dataSize:     56,
		count:        4,
		currentCount: 0,
		interval:     time.Millisecond * 10,
		timeout:      time.Second * 1,
	}
	for _, o := range opts {
		o.apply(m)
	}
	ips, err := net.LookupIP(target)
	if err != nil {
		return nil, err
	}
	m.targetTTL = m.maxTTL + 1
	m.result = make([]*seqResult, m.maxTTL+1, m.maxTTL+1)

	var (
		ip   net.IP
		mode Mode
	)

	for _, IP := range ips {
		if IsIPv4(IP) && !m.forceIPv6 {
			ip = IP
			mode = IPV4Mode
			sa1 := &syscall.SockaddrInet4{}
			copy(sa1.Addr[:], ip.To4())
			m.sa = sa1
			break
		} else if IsIPv6(IP) && !m.forceIPv4 {
			ip = IP
			sa1 := &syscall.SockaddrInet6{}
			copy(sa1.Addr[:], ip.To16())
			m.sa = sa1
			mode = IPV6Mode
			break
		}
	}
	if ip == nil {
		return nil, errors.New("there is not A or AAAA record")
	}
	m.ip = ip
	m.mode = mode
	m.socketFd, err = m.listen(mode)
	if err != nil {
		return nil, err
	}
	return m, nil

}

func (m *GMtr) Run() (*MtrResult, error) {
	var waitTime time.Duration
	for m.count > m.currentCount {
		ld := time.Now().Sub(m.lastSendTime)
		if ld < m.interval {
			// 前一个包发送的间隔没到
			waitTime = m.interval - ld
			goto waitForReply
		}
		m.lastSendTime = time.Now()
		if m.mode == IPV6Mode {
			m.sendV6()
		} else {
			m.sendV4()
		}
		waitTime = m.interval
		m.currentTTL += 1

		if _, ok := m.lastTTL[m.currentCount]; ok {
			// todo 当前这轮ping已经到达对端了
			m.currentTTL = 1
			m.currentCount += 1
		} else if m.currentTTL > m.maxTTL {
			// todo 已经到上限，发送下一轮ping
			m.currentTTL = 1
			m.currentCount += 1
		}

	waitForReply:
		// todo 处理接收的时间
		if m.waitForReply(waitTime) {
			// 如果已经接收到一个数据，继续接收
			for m.waitForReply(0) {
			}
		}
	}

	// todo 发送完成等所有数据回来，或者超时
	for m.evFirst != nil {
		e := m.evDequeue()
		waitTime = e.evTime.Sub(time.Now())
		if waitTime < 0 {
			// todo 接收超时了
			continue
		}
		if m.waitForReply(waitTime) {
			for m.waitForReply(0) {
			}
		}
	}

	allLoss := true
	if m.targetTTL > m.maxTTL {
		// todo 丢包率100%
		for i := len(m.result) - 1; i > 0; i-- {
			im := m.result[i]
			if im == nil {
				continue
			}
			if im.reply != 0 {
				// todo 最后一个接收到的包
				m.targetTTL = i + 1
				allLoss = false
				break
			}
		}
	}
	if allLoss && m.targetTTL > m.maxTTL {
		m.targetTTL = 1
	}
	result := &MtrResult{
		LocalIp:  m.localIp.String(),
		TargetIp: m.ip.String(),
		Mode:     int(m.mode),
	}
	for i := 1; i <= m.targetTTL && i <= m.maxTTL; i++ {
		im := m.result[i]
		if im == nil {
			continue
		}
		r := &MtrTTLResult{
			TTL:   i,
			Count: im.count,
			Loss:  im.loss(),
			Avg:   im.avg(),
			Max:   float64(im.max),
			Min:   float64(im.min),
			Ips:   make(StringSet),
		}
		for ip := range m.result[i].ips {
			r.Ips.Add(ip)
		}
		result.TTL = append(result.TTL, r)
	}
	return result, nil
}

func (m *GMtr) listen(mode Mode) (int, error) {
	var (
		proto  int
		domain int
	)
	switch mode {
	case IPV4Mode:
		proto = syscall.IPPROTO_ICMP
		domain = syscall.AF_INET
	case IPV6Mode:
		proto = syscall.IPPROTO_ICMPV6
		domain = syscall.AF_INET6
	default:
		return 0, fmt.Errorf("unexpected proto %v", m)
	}
	sock, err := syscall.Socket(
		domain,
		syscall.SOCK_RAW,
		proto,
	)
	if err != nil {
		return 0, err
	}
	if err := unix.SetNonblock(sock, true); err != nil {
		unix.Close(sock)
		return 0, err
	}
	if mode == IPV4Mode {
		// todo 设定header自定义
		err = ipv42.HeaderPrepend(sock)
		if err != nil {
			unix.Close(sock)
			return 0, err
		}
	}
	return sock, nil
}

func (m *GMtr) Close() error {
	if m.socketFd > 0 {
		return unix.Close(m.socketFd)
	}
	return nil
}

func (m *GMtr) evRemove(h *seqValue) {
	if m.evFirst == h {
		m.evFirst = h.evNext
	}
	if m.evLast == h {
		m.evLast = h.evPrev
	}
	if h.evPrev != nil {
		h.evPrev.evNext = h.evNext
	}
	if h.evNext != nil {
		h.evNext.evPrev = h.evPrev
	}
	h.evPrev = nil
	h.evNext = nil
}

func (m *GMtr) evDequeue() *seqValue {
	var h *seqValue

	if m.evFirst == nil {
		return nil
	}
	h = m.evFirst
	m.evRemove(h)
	return h
}

func (m *GMtr) evEnqueue(h *seqValue) {
	var i *seqValue
	var iPrev *seqValue
	/* Empty list */
	if m.evLast == nil {
		h.evNext = nil
		h.evPrev = nil
		m.evFirst = h
		m.evLast = h
		return
	}
	if h.evTime.After(m.evLast.evTime) {
		h.evNext = nil
		h.evPrev = m.evLast
		m.evLast.evNext = h
		m.evLast = h
		return
	}
	i = m.evLast
	for {
		iPrev = i.evPrev
		if iPrev == nil || h.evTime.After(iPrev.evTime) {
			h.evPrev = iPrev
			h.evNext = i
			i.evPrev = h
			if iPrev != nil {
				iPrev.evNext = h
			} else {
				m.evFirst = h
			}
			return
		}
		i = iPrev
	}
}
