package client

import (
	"encoding/hex"
	log "github.com/githubDante/go-solarman-proxy/logging"
	"github.com/githubDante/go-solarman-proxy/protocol"
	"net"
	"sync/atomic"
	"time"
)

var clientId uint32 = 0

func nextId() uint32 {
	return atomic.AddUint32(&clientId, 1)
}

type CommSolarman struct {
	Serial uint32
	Client *ClientSolarman
}

// ClientSolarman - А client (e.g. from PySolarmanV5) connected to the proxy
type ClientSolarman struct {
	Conn   net.Conn
	Serial uint32
	Logger *ClientLogger
	// Serial number reporter
	SReport   chan *CommSolarman
	broadcast chan []byte
	Running   bool
	Id        uint32
}

func NewSolarmanClient(conn net.Conn, serialRcv chan *CommSolarman, broadcast chan []byte) *ClientSolarman {
	return &ClientSolarman{
		Conn:      conn,
		SReport:   serialRcv,
		broadcast: broadcast,
		Running:   false,
		Id:        nextId(),
	}
}

func (s *ClientSolarman) Run() {
	s.Running = true
	defer func() {
		s.Running = false
	}()
	for {
		if s.Serial == 0 {
			// The solarman client should send data in 1 minute, otherwise will be disconnected
			s.Conn.SetReadDeadline(time.Now().Add(1 * time.Minute))
		} else {
			s.Conn.SetReadDeadline(time.Time{})
		}
		buffer := make([]byte, 4096)
		log.LogDebugf("Client <%p> waiting for data...\n", s)
		pLen, err := s.Conn.Read(buffer)
		if err != nil || pLen == 0 {
			log.LogErrorf("Client read error: %s\n", err.Error())
			s.Conn.Close()
			return
		}
		if s.Serial == 0 {
			packet, err := protocol.NewV5Frame(buffer[:pLen])
			if err == nil {
				s.Serial = packet.LoggerSN()
				log.LogWarnf("Client [%s] will use serial number <%d>\n", s.Conn.RemoteAddr().String(), s.Serial)
				s.SReport <- &CommSolarman{
					Serial: s.Serial,
					Client: s,
				}
				time.Sleep(5 * time.Millisecond) // for logger association
			} else {
				log.LogErrorf("Bad packet from client <%p>... Forwarding refused.\n", s)
				continue
			}
		}
		if s.Logger != nil {
			//s.Logger.Conn.Write(buffer[:pLen])
			log.LogDebugf("Client <%p> sending data: %s\n", s, hex.EncodeToString(buffer[:pLen]))
			s.Logger.Send(buffer[:pLen], s)
		} else {
			log.LogDebugf("Client <%p> has no logger. Broadcasting data: %s\n",
				s, hex.EncodeToString(buffer[:pLen]))
			s.broadcast <- buffer[:pLen]
		}
	}
}

// Stop closes the client connection
func (s *ClientSolarman) Stop() {
	if s.Running {
		_, _ = s.Conn.Write([]byte{})
	}
	_ = s.Conn.Close()
}

func (s *ClientSolarman) AddLogger(l *ClientLogger) {
	log.LogWarnf("Adding <%p> as logger to <%p>\n", l, s)
	s.Logger = l
}

// Send will send data to connected client
//
// Extra operations can be performed too. Currently only for logging/debug
func (s *ClientSolarman) Send(data []byte) error {
	log.LogDebugf("Client <%p> sending data from <%p>\n", s, s.Logger)
	_, err := s.Conn.Write(data)
	if err != nil {
		log.LogErrorf("Client send error <%s>:  %s\n", s.Conn.RemoteAddr().String(), err.Error())
	}
	return err
}
