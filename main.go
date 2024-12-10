package main

import (
	"flag"
	"fmt"
	log "github.com/githubDante/go-solarman-proxy/logging"
	"os"
	"strconv"

	"github.com/githubDante/go-solarman-proxy/server"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "%s [flags] <IP Address> <Port>\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "Flags:\n")
		flag.PrintDefaults()
	}
	debug := flag.Bool("debug", false, "enable debug logging")
	silent := flag.Bool("silent", false, "enable silent mode")
	bcast := flag.Bool("bcast", false, "enable the broadcast listener")
	buffer := flag.Bool("buffered", false, "enable the logger write buffer (sequential client communication)")
	flag.Parse()
	args := flag.Args()

	if len(args) < 2 {
		log.LogErrorf("[%s] requires IPAddress and port\n", os.Args[0])
		os.Exit(1)
	}

	ip := args[0]
	port, err := strconv.ParseUint(args[1], 10, 64)
	if err != nil || port > 65535 {
		log.LogErrorf("[%s] port error...\n", os.Args[0])
		os.Exit(1)
	}
	if *debug {
		log.EnableDebug()
	}
	if *silent {
		log.EnableSilent()
	}
	proxy := server.NewProxy(ip, int(port))
	err = proxy.Serve(*bcast, *buffer)
	if err != nil {
		log.LogErrorf("Proxy start error: %s\n", err.Error())
		os.Exit(1)
	}

	proxy.Wait()
}
