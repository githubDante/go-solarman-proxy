package main

import (
	"flag"
	log "github.com/githubDante/go-solarman-proxy/logging"
	"os"
	"strconv"

	"github.com/githubDante/go-solarman-proxy/server"
)

func main() {
	debug := flag.Bool("debug", false, "enable debug logging")
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
	proxy := server.NewProxy(ip, int(port))
	err = proxy.Serve()
	if err != nil {
		log.LogErrorf("Proxy start error: %s\n", err.Error())
		os.Exit(1)
	}

	proxy.Wait()
}
