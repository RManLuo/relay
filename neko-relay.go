package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"neko-relay/relay"
	"neko-relay/stat"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

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
func check(r Rule) (bool, error) {
	if r.Port > 65535 || r.Rport > 65535 {
		return false, errors.New("port is not in range")
	}
	return true, nil
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
	var r = Rule{Port: Port, Remote: remote, RIP: RIP, Rport: Rport, Type: typ}
	passed, err := check(r)
	if !passed {
		return
	}
	Rules[rid] = r
	_, has := Traffic[rid]
	if !has {
		Traffic[rid] = relay.NewTF()
	}
	return
}
func main() {
	flag.Parse()
	if *show_version != false {
		fmt.Println("neko-relay v1.2")
		fmt.Println("TCP & UDP & WSTUNNEL & STAT")
		return
	}
	if *debug != true {
		gin.SetMode(gin.ReleaseMode)
	}
	relay.GetCert()
	r := gin.New()
	r.GET("/data/"+*key, func(c *gin.Context) {
		// fmt.Println(Rules, Traffic, tcp_lis)
		c.JSON(200, gin.H{"Rules": Rules, "Svrs": Svrs, "Traffic": Traffic})
	})
	if *debug != true {
		r.Use(webMiddleware)
	}
	r.POST("/traffic", func(c *gin.Context) {
		reset, _ := strconv.ParseBool(c.DefaultPostForm("reset", "false"))
		y := gin.H{}
		for rid, tf := range Traffic {
			y[rid] = tf.Total()
			if reset {
				_, has := Rules[rid]
				if has {
					tf.Reset()
				} else {
					delete(Traffic, rid)
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
		delete(Rules, rid)
		resp(c, true, nil, 200)
	})
	r.POST("/sync", func(c *gin.Context) {
		newRules := make(map[string]Rule)
		json.Unmarshal([]byte(c.PostForm("rules")), &newRules)
		for rid, r := range newRules {
			rip, err := getIP(r.Remote)
			if err == nil {
				nr := Rule{Port: r.Port, Remote: r.Remote, RIP: rip, Rport: r.Rport, Type: r.Type}
				pass, _ := check(nr)
				if pass {
					newRules[rid] = nr
				} else {
					delete(newRules, rid)
				}
			} else {
				delete(newRules, rid)
			}
		}
		if *debug {
			fmt.Println(newRules)
		}
		for rid := range Rules {
			rule, has := newRules[rid]
			if has && rule == Rules[rid] {
				delete(newRules, rid)
			} else {
				del(rid)
				time.Sleep(1 * time.Millisecond)
				delete(Rules, rid)
			}
		}
		for rid, rule := range newRules {
			if *debug {
				fmt.Println(rule)
			}
			Rules[rid] = rule
			_, has := Traffic[rid]
			if !has {
				Traffic[rid] = relay.NewTF()
			}
			go add(rid)
			time.Sleep(5 * time.Millisecond)
		}
		resp(c, true, Rules, 200)
	})

	r.GET("/stat", func(c *gin.Context) {
		res, err := stat.GetStat()
		if err == nil {
			resp(c, true, res, 200)
		} else {
			resp(c, false, err, 500)
		}
	})

	// Rules["test_iperf3"] = Rule{
	// 	Port:   5202,
	// 	Remote: "127.0.0.1",
	// 	RIP:    "127.0.0.1",
	// 	Rport:  uint(5201),
	// 	Type:   "tcp+udp",
	// }
	// go add("test_iperf3")
	// time.Sleep(10 * time.Millisecond)
	// Rules["test_server"] = Rule{
	// 	Port:   3333,
	// 	Remote: "127.0.0.1",
	// 	RIP:    "127.0.0.1",
	// 	Rport:  uint(5201),
	// 	Type:   "wss_tunnel_server",
	// }
	// go add("test_server")
	// time.Sleep(10 * time.Millisecond)
	// Rules["test_client"] = Rule{
	// 	Port:   4444,
	// 	Remote: "127.0.0.1",
	// 	RIP:    "127.0.0.1",
	// 	Rport:  uint(3333),
	// 	Type:   "wss_tunnel_client",
	// }
	// go add("test_client")
	// time.Sleep(10 * time.Millisecond)

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
