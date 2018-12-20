/**
*FileName: master
*Create on 2018-12-18 16:22
*Create by mok
 */

package main

import (
	"crontab/master/config"
	"crontab/master/router"
	"flag"
	"github.com/julienschmidt/httprouter"
	"log"
	"net/http"
	"runtime"
	"time"
)

var confpath string

func init() {
	flag.StringVar(&confpath, "f", "./config/conf.json", "config file path")
}
func main() {
	flag.Parse()
	var err error
	if err = config.Init(confpath); err != nil {
		log.Fatal(err)
	}
	conf := config.Conf
	runtime.GOMAXPROCS(int(conf.Cpu))
	r := httprouter.New()
	router.LoadRouter(r)
	server := http.Server{
		Addr:         conf.Addr,
		ReadTimeout:  time.Duration(conf.ReadTimeOut) * time.Millisecond,
		WriteTimeout: time.Duration(conf.WriteTimeOut) * time.Millisecond,
		Handler:      r,
	}

	if err = server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
