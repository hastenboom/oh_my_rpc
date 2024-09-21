package HastenClient

import "sync"

type Balancer interface {
	GetNextIp() string
}

type Strategy int

const (
	Round Strategy = iota
)

func BalancerFactory(strategy Strategy, ipList []string) Balancer {
	switch strategy {
	case Round:
		return NewBalancer(ipList)
	default:
		panic("Invalid strategy")
	}

}

type RoundBalancer struct {
	IpList []string
	Index  int
	lock   sync.Mutex
}

func NewBalancer(ipList []string) *RoundBalancer {
	return &RoundBalancer{
		IpList: ipList,
		Index:  0,
		lock:   sync.Mutex{},
	}
}

func (b *RoundBalancer) GetNextIp() string {
	b.lock.Lock()
	defer b.lock.Unlock()
	ip := b.IpList[b.Index]
	b.Index = (b.Index + 1) % len(b.IpList)
	return ip
}
