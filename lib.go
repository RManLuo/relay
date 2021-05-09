package main

import (
	"neko-relay/relay"
	"net"
	"strconv"
	"time"
)

type Rule struct {
	Port   uint   `json:port`
	Remote string `json:remote`
	RIP    string
	Rport  uint   `json:rport`
	Type   string `json:type`
}

var (
	Rules   = make(map[string]Rule)
	Traffic = make(map[string]*relay.TF)
	Svrs    = make(map[string]*relay.Relay)
)

func add(rid string) (err error) {
	r := Rules[rid]
	local_addr := ":" + strconv.Itoa(int(r.Port))
	remote_addr := r.RIP + ":" + strconv.Itoa(int(r.Rport))
	_, has := Traffic[rid]
	if !has {
		Traffic[rid] = relay.NewTF()
	}
	Svrs[rid], err = relay.NewRelay(local_addr, remote_addr, 30, 10, Traffic[rid], r.Type)
	Svrs[rid].ListenAndServe()
	// fmt.Println(local_addr, "<=>", remote_addr)

	// if strings.Contains(r.Type, "tcp") {
	// 	add_tcp(rid, local_addr, remote_addr)
	// }
	// if strings.Contains(r.Type, "udp") {
	// 	add_udp(rid, local_addr, remote_addr)
	// }
	return
}
func del(rid string) {
	Svr, has := Svrs[rid]
	if has {
		Svr.Shutdown()
		time.Sleep(100 * time.Millisecond)
		delete(Svrs, rid)
	}
}

func getIP(host string) (ip string, err error) {
	ips, err := net.LookupHost(host)
	if err != nil {
		return
	}
	ip = ips[0]
	return
}

func ddns() {
	for {
		time.Sleep(time.Second * 60)
		for rid, rule := range Rules {
			RIP, err := getIP(rule.Remote)
			if err == nil && RIP != rule.RIP {
				rule.RIP = RIP
				del(rid)
				go add(rid)
			}
		}
	}
}
