package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type tf struct {
	tcp_up   uint64
	tcp_down uint64
	udp_up   uint64
	udp_down uint64
	rw       *sync.RWMutex
}
type rule struct {
	Port   uint   `json:port`
	Remote string `json:remote`
	RIP    string
	Rport  uint   `json:rport`
	Type   string `json:type`
}

var (
	rules      = make(map[string]rule)
	traffic    = make(map[string]*tf)
	tcp_lis    = make(map[string]net.Listener)
	tcp_remote = make(map[string]net.Listener)

	POOL = sync.Pool{
		New: func() interface{} {
			return make([]byte, 32*1024)
		},
	}
)

func fw_tcp(rid string, dst io.Writer, src io.Reader, reverse bool) {
	buf := POOL.Get().([]byte)
	defer POOL.Put(buf)
	for {
		if src == nil || dst == nil {
			return
		}
		n, err := src.Read(buf[:])
		if err != nil {
			return
		}
		if _, err := dst.Write(buf[0:n]); err != nil {
			return
		}
		bytes := uint64(n)
		// fmt.Println(bytes)
		if err != nil {
			fmt.Println(err)
			// return
		}
		traffic[rid].rw.Lock()
		if reverse {
			traffic[rid].tcp_down += bytes
		} else {
			traffic[rid].tcp_up += bytes
		}
		traffic[rid].rw.Unlock()
	}
}
func add_tcp(rid string, local_addr string, remote_addr string) (err error) {
	// fmt.Println(local_addr, "<=>", remote_addr)
	tcp_lis[rid], err = net.Listen("tcp", local_addr)
	if err != nil {
		fmt.Println("tcp_listen:", err)
		return
	}
	defer tcp_lis[rid].Close()
	defer delete(tcp_lis, rid)
	for {
		lis, has := tcp_lis[rid]
		if !has {
			return
		}
		local_tcp, err1 := lis.Accept() //接受tcp客户端连接，并返回新的套接字进行通信
		if err1 != nil {
			return
		}
		remote_tcp, err2 := net.Dial("tcp", remote_addr) //连接目标服务器
		if err2 != nil {
			continue
		}
		go fw_tcp(rid, local_tcp, remote_tcp, false)
		go fw_tcp(rid, remote_tcp, local_tcp, true)
	}
}

func add(rid string) {
	r := rules[rid]
	local_addr := ":" + strconv.Itoa(int(r.Port))
	remote_addr := r.RIP + ":" + strconv.Itoa(int(r.Rport))
	// fmt.Println(local_addr, "<=>", remote_addr)
	if strings.Contains(r.Type, "tcp") {
		add_tcp(rid, local_addr, remote_addr)
	}
	// if strings.Contains(r.Type, "udp") {
	// 	add_udp(rid, local_addr, remote_addr)
	// }
}
func del(rid string) {
	lis, has := tcp_lis[rid]
	if has {
		lis.Close()
		delete(tcp_lis, rid)
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
func newTf() *tf {
	return &tf{tcp_up: 0, tcp_down: 0, udp_up: 0, udp_down: 0, rw: new(sync.RWMutex)}
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
		traffic[rid] = newTf()
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
	r.Use(webMiddleware)
	r.GET("/data", func(c *gin.Context) {
		fmt.Println(rules, traffic, tcp_lis)
		c.JSON(200, gin.H{"rules": rules, "tcp": tcp_lis, "traffic": traffic})
	})
	r.POST("/traffic", func(c *gin.Context) {
		reset, _ := strconv.ParseBool(c.DefaultPostForm("reset", "false"))
		y := gin.H{}
		for rid, x := range traffic {
			x.rw.RLock()
			y[rid] = x.tcp_up + x.tcp_down + x.udp_up + x.udp_down
			x.rw.RUnlock()
		}
		if reset {
			traffic = make(map[string]*tf)
			for rid := range rules {
				traffic[rid] = newTf()
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
			rip, err := getIP(r.Remote)
			if err == nil {
				newRules[rid] = rule{Port: r.Port, Remote: r.Remote, RIP: rip, Rport: r.Rport, Type: r.Type}
			} else {
				delete(newRules, rid)
			}
		}
		for rid := range rules {
			rule, has := newRules[rid]
			if has && rule == rules[rid] {
				delete(newRules, rid)
			} else {
				del(rid)
				delete(rules, rid)
			}
		}
		for rid, rule := range newRules {
			rules[rid] = rule
			traffic[rid] = newTf()
			go add(rid)
		}
		resp(c, true, nil, 200)
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
