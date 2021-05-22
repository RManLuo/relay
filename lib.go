package main

import (
	"fmt"
	"neko-relay/relay"
	"net"
	"strconv"
	"time"

	cmap "github.com/orcaman/concurrent-map"
)

type Rule struct {
	Port   uint   `json:port`
	Remote string `json:remote`
	RIP    string
	Rport  uint   `json:rport`
	Type   string `json:type`
}

var (
	Rules   = cmap.New()
	Traffic = cmap.New()
	Svrs    = cmap.New()
)

func getTF(rid string) (tf *relay.TF) {
	Tf, hast := Traffic.Get(rid)
	if hast {
		tf = Tf.(*relay.TF)
	}
	if !hast {
		tf = relay.NewTF()
		Traffic.Set(rid, tf)
	}
	return
}

func start(rid string) (err error) {
	rule, hasr := Rules.Get(rid)
	if !hasr {
		return
	}
	r := rule.(Rule)
	local_addr := ":" + strconv.Itoa(int(r.Port))
	remote_addr := r.RIP + ":" + strconv.Itoa(int(r.Rport))

	svr, err := relay.NewRelay(local_addr, remote_addr, r.RIP, 30, 10, getTF(rid), r.Type)
	if err != nil {
		return
	}
	Svrs.Set(rid, svr)
	svr.Serve()
	return
}
func stop(rid string) {
	Svr, has := Svrs.Get(rid)
	if has {
		Svr.(*relay.Relay).Close()
		// time.Sleep(10 * time.Millisecond)
		Svrs.Remove(rid)
	}
}
func cmp(x, y Rule) bool {
	return x.Port == y.Port && x.Remote == y.Remote && x.Rport == y.Rport && x.Type == y.Type
}
func sync(newRules map[string]Rule) {
	if config.Debug {
		fmt.Println(newRules)
	}
	for item := range Rules.Iter() {
		rid := item.Key
		rule, has := newRules[rid]
		if has && cmp(rule, item.Val.(Rule)) {
			delete(newRules, rid)
		} else {
			stop(rid)
			time.Sleep(1 * time.Millisecond)
			Rules.Remove(rid)
		}
	}
	for rid, rule := range newRules {
		if config.Debug {
			fmt.Println(rule)
		}
		rip, err := getIP(rule.Remote)
		if err != nil {
			continue
		}
		rule.RIP = rip
		pass, _ := check(rule)
		if !pass {
			continue
		}
		Rules.Set(rid, rule)
		go start(rid)
		time.Sleep(5 * time.Millisecond)
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
		for item := range Rules.Iter() {
			rid, rule := item.Key, item.Val.(Rule)
			RIP, err := getIP(rule.Remote)
			if err == nil && RIP != rule.RIP {
				rule.RIP = RIP
				stop(rid)
				go start(rid)
			}
		}
	}
}
