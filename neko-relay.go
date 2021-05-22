package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"neko-relay/relay"
	"neko-relay/stat"
	"strconv"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v2"
)

type CONF struct {
	Key      string
	Port     int
	Debug    bool
	Certfile string
	Keyfile  string
	Syncfile string
}

var config CONF

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
	Rules.Set(rid, r)
	return
}
func main() {
	var confpath string
	var show_version bool
	flag.StringVar(&confpath, "c", "", "config")
	flag.StringVar(&config.Key, "key", "key", "api key")
	flag.IntVar(&config.Port, "port", 8080, "api port")
	flag.BoolVar(&config.Debug, "config.Debug", false, "enable config.Debug")
	flag.StringVar(&config.Certfile, "certfile", "public.pem", "cert file")
	flag.StringVar(&config.Keyfile, "keyfile", "private.key", "key file")
	flag.StringVar(&config.Syncfile, "syncfile", "", "sync file")
	flag.BoolVar(&show_version, "v", false, "show version")
	flag.Parse()
	if confpath != "" {
		data, err := ioutil.ReadFile(confpath)
		if err != nil {
			log.Panic(err)
		}
		err = yaml.Unmarshal([]byte(data), &config)
		if err != nil {
			panic(err)
		}
		// fmt.Println(config)
	}
	if show_version != false {
		fmt.Println("neko-relay v1.3")
		fmt.Println("TCP & UDP & WS TUNNEL && WSS TUNNEL & STAT")
		return
	}
	if config.Debug != true {
		gin.SetMode(gin.ReleaseMode)
	}
	relay.CertFile = config.Certfile
	relay.KeyFile = config.Keyfile
	relay.GetCert()
	r := gin.New()
	r.GET("/data/"+config.Key, func(c *gin.Context) {
		c.JSON(200, gin.H{"Rules": Rules, "Svrs": Svrs, "Traffic": Traffic})
	})
	if config.Debug != true {
		r.Use(webMiddleware)
	}
	r.POST("/traffic", func(c *gin.Context) {
		reset, _ := strconv.ParseBool(c.DefaultPostForm("reset", "false"))
		y := gin.H{}
		for item := range Traffic.Iter() {
			rid, tf := item.Key, item.Val.(*relay.TF)
			y[rid] = tf.Total()
			if reset {
				tf.Reset()
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
		go start(rid)
		resp(c, true, nil, 200)
	})
	r.POST("/edit", func(c *gin.Context) {
		rid, err := ParseRule(c)
		if err != nil {
			resp(c, false, err.Error(), 500)
			return
		}
		stop(rid)
		go start(rid)
		resp(c, true, nil, 200)
	})
	r.POST("/del", func(c *gin.Context) {
		rid := c.PostForm("rid")
		rule, has := Rules.Get(rid)
		if !has {
			resp(c, false, gin.H{
				"rule":    nil,
				"traffic": 0,
			}, 200)
			return
		}
		traffic, _ := Traffic.Get(rid)
		stop(rid)
		Rules.Remove(rid)
		Traffic.Remove(rid)
		resp(c, true, gin.H{
			"rule":    rule,
			"traffic": traffic,
		}, 200)
	})
	r.POST("/sync", func(c *gin.Context) {
		newRules := make(map[string]Rule)
		data := []byte(c.PostForm("rules"))
		json.Unmarshal(data, &newRules)
		if config.Syncfile != "" {
			err := ioutil.WriteFile(config.Syncfile, data, 0644)
			if err != nil {
				log.Println(err)
			}
		}
		sync(newRules)
		resp(c, true, Rules, 200)
	})

	if config.Syncfile != "" {
		data, err := ioutil.ReadFile(config.Syncfile)
		if err == nil {
			newRules := make(map[string]Rule)
			json.Unmarshal(data, &newRules)
			sync(newRules)
		} else {
			log.Println(err)
		}
	}

	r.GET("/stat", func(c *gin.Context) {
		res, err := stat.GetStat()
		if err == nil {
			resp(c, true, res, 200)
		} else {
			resp(c, false, err, 500)
		}
	})
	go ddns()
	fmt.Println("Api port:", config.Port)
	fmt.Println("Api key:", config.Key)
	r.Run(":" + strconv.Itoa(config.Port))
}
func webMiddleware(c *gin.Context) {
	if c.Request.Header.Get("key") != config.Key {
		resp(c, false, "Api key Incorrect", 500)
		c.Abort()
		return
	}
	c.Next()
}
