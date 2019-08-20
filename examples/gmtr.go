package main

import (
	"fmt"
	"github.com/neo-hu/gmtr"
	"time"
)

func main() {
	m, err := gmtr.NewGMtr("www.sina.com.cn")
	if err != nil {
		fmt.Println(err)
		return
	}
	result, err := m.Run()
	if err != nil {
		fmt.Println(err)
		return
	}
	for ttl, val := range result.TTL {
		fmt.Printf("ttl:%d, ip:%v loss:%v%%, avg:%v, max:%v, min:%v\n", ttl+1, val.Ips, val.Loss, time.Duration(val.Avg), time.Duration(val.Max), time.Duration(val.Min))
	}
}
