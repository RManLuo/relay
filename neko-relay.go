package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"neko-relay/relay"
	"net"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type rule struct {
	Port   uint   `json:port`
	Remote string `json:remote`
	RIP    string
	Rport  uint   `json:rport`
	Type   string `json:type`
}

var (
	rules   = make(map[string]rule)
	traffic = make(map[string]*relay.TF)
	svrs    = make(map[string]*relay.Relay)
)

func add(rid string) (err error) {
	r := rules[rid]
	local_addr := ":" + strconv.Itoa(int(r.Port))
	remote_addr := r.RIP + ":" + strconv.Itoa(int(r.Rport))
	traffic[rid] = relay.NewTF()
	svrs[rid], err = relay.NewRelay(local_addr, remote_addr, 100, 100, traffic[rid])
	svrs[rid].ListenAndServe()
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
	svr, has := svrs[rid]
	if has {
		svr.Shutdown()
		time.Sleep(100 * time.Millisecond)
		delete(svrs, rid)
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
		for rid, rule := range rules {
			RIP, err := getIP(rule.Remote)
			if err == nil && RIP != rule.RIP {
				rule.RIP = RIP
				del(rid)
				go add(rid)
			}
		}
	}
}

var (
	key          = flag.String("key", "key", "api key")
	port         = flag.String("port", "8080", "api port")
	debug        = flag.Bool("debug", false, "enable debug")
	show_version = flag.Bool("version", false, "show version")
)

func resp(c *gin.Context, success bool, data interface{}, code int) {
	c.JSON(code, gin.H{
		"success": success,
		"data":    data,
	})
}
func ParseRule(c *gin.Context) (rid string, err error) {
	rid = c.PostForm("rid")
	port, _ := strconv.Atoi(c.PostForm("port"))
	remote := c.PostForm("remote")
	rport, _ := strconv.Atoi(c.PostForm("rport"))
	typ := c.PostForm("type")
	RIP, err := getIP(remote)
	if err != nil {
		return
	}
	rules[rid] = rule{Port: uint(port), Remote: remote, RIP: RIP, Rport: uint(rport), Type: typ}
	_, has := traffic[rid]
	if !has {
		traffic[rid] = relay.NewTF()
	}
	return
}
func main() {
	flag.Parse()
	if *show_version != false {
		fmt.Println("neko-relay v1.0")
		return
	}
	if *debug != true {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.GET("/data/"+*key, func(c *gin.Context) {
		// fmt.Println(rules, traffic, tcp_lis)
		c.JSON(200, gin.H{"rules": rules, "svrs": svrs, "traffic": traffic})
	})
	r.Use(webMiddleware)
	r.POST("/traffic", func(c *gin.Context) {
		reset, _ := strconv.ParseBool(c.DefaultPostForm("reset", "false"))
		y := gin.H{}
		for rid, x := range traffic {
			x.RW.RLock()
			y[rid] = x.TCP_UP + x.TCP_DOWN + x.UDP
			if reset {
				_, has := rules[rid]
				if has {
					x.TCP_UP = 0
					x.TCP_DOWN = 0
					x.UDP = 0
				} else {
					delete(traffic, rid)
				}
			}
			x.RW.RUnlock()
		}
		resp(c, true, y, 200)
	})
	r.POST("/add", func(c *gin.Context) {
		rid, err := ParseRule(c)
		if err != nil {
			resp(c, false, err.Error(), 500)
			return
		}
		go add(rid)
		resp(c, true, nil, 200)
	})
	r.POST("/edit", func(c *gin.Context) {
		rid, err := ParseRule(c)
		if err != nil {
			resp(c, false, err.Error(), 500)
			return
		}
		del(rid)
		go add(rid)
		resp(c, true, nil, 200)
	})
	r.POST("/del", func(c *gin.Context) {
		rid := c.PostForm("rid")
		del(rid)
		delete(rules, rid)
		resp(c, true, nil, 200)
	})
	r.POST("/sync", func(c *gin.Context) {
		newRules := make(map[string]rule)
		json.Unmarshal([]byte(c.PostForm("rules")), &newRules)
		for rid, r := range newRules {
			rip, err := getIP(r.Remote)
			if err == nil {
				newRules[rid] = rule{Port: r.Port, Remote: r.Remote, RIP: rip, Rport: r.Rport, Type: r.Type}
			} else {
				delete(newRules, rid)
			}
		}
		if *debug {
			fmt.Println(newRules)
		}
		for rid := range rules {
			rule, has := newRules[rid]
			if has && rule == rules[rid] {
				delete(newRules, rid)
			} else {
				del(rid)
				time.Sleep(5 * time.Millisecond)
				delete(rules, rid)
			}
		}
		for rid, rule := range newRules {
			if *debug {
				fmt.Println(rule)
			}
			rules[rid] = rule
			traffic[rid] = relay.NewTF()
			go add(rid)
			time.Sleep(20 * time.Millisecond)
		}
		resp(c, true, rules, 200)
	})
	go ddns()
	fmt.Println("Api port:", *port)
	fmt.Println("Api key:", *key)
	r.Run(":" + *port)
}
func webMiddleware(c *gin.Context) {
	if c.Request.Header.Get("key") != *key {
		resp(c, false, "Api key Incorrect", 500)
		c.Abort()
		return
	}
	c.Next()
}
