package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"neko-relay/relay"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	gnet "github.com/shirou/gopsutil/net"

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
	svrs[rid], err = relay.NewRelay(local_addr, remote_addr, 30, 10, traffic[rid])
	svrs[rid].ListenAndServe(
		strings.Contains(r.Type, "tcp"),
		strings.Contains(r.Type, "udp"),
		strings.Contains(r.Type, "websocket"),
		strings.Contains(r.Type, "tls"),
	)
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
	Port := uint(port)
	remote := c.PostForm("remote")
	rport, _ := strconv.Atoi(c.PostForm("rport"))
	Rport := uint(rport)
	typ := c.PostForm("type")
	RIP, err := getIP(remote)
	if err != nil {
		return
	}
	if Port < 0 || Port > 65535 || Rport < 0 || Rport > 65535 {
		err = errors.New("port is not in range")
		return
	}
	rules[rid] = rule{Port: Port, Remote: remote, RIP: RIP, Rport: Rport, Type: typ}
	_, has := traffic[rid]
	if !has {
		traffic[rid] = relay.NewTF()
	}
	return
}
func main() {
	flag.Parse()
	if *show_version != false {
		fmt.Println("neko-relay v1.1")
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
	if *debug != true {
		r.Use(webMiddleware)
	}
	r.POST("/traffic", func(c *gin.Context) {
		reset, _ := strconv.ParseBool(c.DefaultPostForm("reset", "false"))
		y := gin.H{}
		for rid, tf := range traffic {
			y[rid] = tf.Total()
			if reset {
				_, has := rules[rid]
				if has {
					tf.Reset()
				} else {
					delete(traffic, rid)
				}
			}
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
			if int(r.Port) < 0 || int(r.Port) > 65535 || int(r.Rport) < 0 || int(r.Rport) > 65535 {
				delete(newRules, rid)
				continue
			}
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
			time.Sleep(30 * time.Millisecond)
		}
		resp(c, true, rules, 200)
	})
	go ddns()

	r.GET("/stat", func(c *gin.Context) {
		CPU1, err := cpu.Times(true)
		if err != nil {
			resp(c, false, nil, 500)
			return
		}
		NET1, err := gnet.IOCounters(true)
		if err != nil {
			resp(c, false, nil, 500)
			return
		}
		time.Sleep(200 * time.Millisecond)
		CPU2, err := cpu.Times(true)
		if err != nil {
			resp(c, false, nil, 500)
			return
		}
		NET2, err := gnet.IOCounters(true)
		if err != nil {
			resp(c, false, nil, 500)
			return
		}
		MEM, err := mem.VirtualMemory()
		if err != nil {
			resp(c, false, nil, 500)
			return
		}
		SWAP, err := mem.SwapMemory()
		if err != nil {
			resp(c, false, nil, 500)
			return
		}
		single := make([]float64, len(CPU1))
		var idle, total, multi float64
		idle, total = 0, 0
		for i, c1 := range CPU1 {
			c2 := CPU2[i]
			single[i] = 1 - (c2.Idle-c1.Idle)/(c2.Total()-c1.Total())
			idle += c2.Idle - c1.Idle
			total += c2.Total() - c1.Total()
		}
		multi = 1 - idle/total
		var in, out, in_total, out_total uint64
		in, out, in_total, out_total = 0, 0, 0, 0
		for i, x := range NET2 {
			if x.Name == "lo" {
				continue
			}
			in += x.BytesRecv - NET1[i].BytesRecv
			out += x.BytesSent - NET1[i].BytesSent
			in_total += x.BytesRecv
			out_total += x.BytesSent
		}
		resp(c, true, gin.H{
			"cpu": gin.H{"multi": multi, "single": single},
			"net": gin.H{
				"delta": gin.H{
					"in":  float64(in) / 0.2,
					"out": float64(out) / 0.2,
				},
				"total": gin.H{
					"in":  in_total,
					"out": out_total,
				},
			},
			"mem": gin.H{
				"virtual": MEM,
				"swap":    SWAP,
			},
		}, 200)
	})

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
