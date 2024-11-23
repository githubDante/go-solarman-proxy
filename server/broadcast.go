package server

import (
	"fmt"
	log "github.com/githubDante/go-solarman-proxy/logging"
	"net"
)

const (
	magic = "WIFIKIT-214028-READ"
	mac   = "563570726f78 " // V5prox as MAC address
)

func (s *V5ProxyServer) handleScanBroadcasts() {
	br, err := net.ListenUDP("udp4", &net.UDPAddr{
		IP:   net.IPv4(0, 0, 0, 0),
		Port: 48899,
	})
	if err != nil {
		log.LogErrorf("Cannot create broadcast listener: %s\n", err.Error())
		return
	}
	log.LogInfof("Broadcast listener created <%s>\n", br.LocalAddr().String())
	defer br.Close()

	for {
		buffer := make([]byte, 4096)
		n, addr, rErr := br.ReadFromUDP(buffer)
		if rErr != nil {
			continue
		}
		log.LogDebugf("Got <%d bytes> broadcast from %s - %s\n", n, addr, string(buffer[:n]))
		if string(buffer[:len(magic)]) == magic {
			log.LogInfof("Sending broadcast reply to <%s:%d>\n", addr.IP.String(), addr.Port)
			for _, logger := range s.loggers {
				if logger != nil && logger.Serial != 0 {
					//ip,mac,serial
					r := fmt.Sprintf("%s,%s,%d", s.Host, mac, logger.Serial)
					_, _ = br.WriteToUDP([]byte(r), addr)
				}
			}
			log.LogDebugf("Broadcast response competed!\n")

		}

	}
}
