package main

import (
	"flag"
	log "github.com/sirupsen/logrus"
)

func main() {
	var myflag string

	flag.StringVar(&myflag, "myflag", "", "my var")
	flag.Parse()

	log.Warnf("this is the value : %s", myflag)
}
