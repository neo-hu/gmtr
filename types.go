package gmtr

import (
	"time"
)

type MtrResult struct {
	LocalIp  string
	TargetIp string
	TTL      []*MtrTTLResult
	Mode     int
}
type MtrTTLResult struct {
	TTL   int
	Loss  float64
	Avg   float64
	Max   float64
	Min   float64
	Ips   StringSet
	Count int
}

type seqValue struct {
	ttl  int
	ping int
	time time.Time

	evPrev *seqValue /* double linked list for the event-queue */
	evNext *seqValue /* double linked list for the event-queue */
	evTime time.Time
}

type seqResult struct {
	max   time.Duration
	sum   time.Duration
	min   time.Duration
	count int
	reply int
	ips   StringSet
}

func (s seqResult) avg() float64 {
	if s.sum > 0 && s.reply > 0 {
		return float64(s.sum) / float64(s.reply)
	}
	return 0

}
func (s seqResult) loss() float64 {
	var loss float64 = 0
	if s.count > 0 {
		loss = float64((s.count-s.reply)*100) / float64(s.count)
	}
	return loss
}
