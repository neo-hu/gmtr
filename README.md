通过更大的采样来跟踪路由，就像 traceroute + ping 命令的组合
<br>
traceroute 通过发送udp包，端口是随机的导致每一跳有可能有多个ip
<br>
mtr发送的是icmp
<br>

```
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

ForceIPv4Option 强制使用ipv4
ForceIPv6Option 强制使用ipv6
DataSizeOption icmp包的大小
CountOption 发送icmp的次数
IntervalOption 发送icmp的间隔，间隔越小越快
TimeoutOption 接收icmp超时的时间
MaxTTLOption 最大的ttl, defaul 30
IdentOption icmp 标记, defaul 本程序的pid

```
