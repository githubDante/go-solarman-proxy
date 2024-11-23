package client

import (
	"encoding/hex"
	log "github.com/githubDante/go-solarman-proxy/logging"
	"net"
	"sync"
	"time"

	"github.com/githubDante/go-solarman-proxy/protocol"
)

// ClientLogger - А data logger connected to the proxy
type ClientLogger struct {
	Conn   net.Conn
	Serial uint32
	// Associated clients
	Clients map[uint32]*ClientSolarman
	lock    sync.Mutex
	// Reporting channel for serial numbers
	SReporter chan *CommLogger
	// Reporting channel on socket disconnection
	stoppedCh chan *CommLogger
	Running   bool
	Id        uint32
}

// NewLoggerClient - Initializes a new data-logger client
//
// Params:
//   - conn: socket
//   - serialRcv: channel for sending the serial number of the data-logger when obtained
//   - disconnectChan: notification channel for closed data-logger socket/stopped read loop
func NewLoggerClient(conn net.Conn, serialRcv chan *CommLogger, disconnectChan chan *CommLogger) *ClientLogger {
	return &ClientLogger{
		Conn:      conn,
		Clients:   make(map[uint32]*ClientSolarman),
		lock:      sync.Mutex{},
		SReporter: serialRcv,
		Running:   false,
		Id:        nextId(),
		stoppedCh: disconnectChan,
	}
}

type CommLogger struct {
	Serial uint32
	Logger *ClientLogger
}

func (c *ClientLogger) Run() {
	c.Running = true
	defer func() {
		c.Running = false
		if c.stoppedCh != nil {
			c.stoppedCh <- &CommLogger{Serial: c.Id, Logger: c}
		}
	}()
	c.serialProbe() // probe for serial on connect
	for {
		buffer := make([]byte, 2048)
		//time.Sleep(200 * time.Millisecond)
		log.LogDebugf("Logger <%p> waiting for data...\n", c)
		pLen, err := c.Conn.Read(buffer)
		if err != nil {
			//fmt.Fprintf(os.Stdout, "Logger <%d> [%s] connection closed?!?\n", c.Serial, c.Conn.RemoteAddr().String())
			log.LogErrorf("<%d> Err?!? - %s\n", pLen, err.Error())
			c.Conn.Close()
			return
		}
		if c.Serial == 0 {
			packet, err := protocol.NewV5Frame(buffer[:pLen])
			if err == nil {
				c.Serial = packet.LoggerSN()
				log.LogDebugf("Logger <%s> provided SN [%d]\n", c.Conn.RemoteAddr().String(), c.Serial)
				c.SReporter <- &CommLogger{Serial: c.Serial, Logger: c}
				time.Sleep(10 * time.Millisecond)
			} else {
				log.LogErrorf("Bad packet from logger <%p>. Cannot create V5 frame from: %s\n",
					c, hex.Dump(buffer[:pLen]))
				continue
			}
		}

		go c.sendToAll(buffer[:pLen])
	}
}

// Stop will close the logger socket
func (c *ClientLogger) Stop() {
	c.Conn.SetDeadline(time.Now().Add(5 * time.Millisecond))
	if !c.Running {
		_ = c.Conn.Close()
	} else {
		_, _ = c.Conn.Write([]byte{0})
		_ = c.Conn.Close()
	}
}

// sendToAll packet broadcast to all connected clients
func (c *ClientLogger) sendToAll(data []byte) {
	c.lock.Lock()
	defer c.lock.Unlock()
	stopped := make([]uint32, 0)
	for _, cl := range c.Clients {
		// Send to all clients
		//_, err := cl.Conn.Write(data)
		if cl.Running {
			err := cl.Send(data)
			if err != nil {
				log.LogWarnf("Client <%p> marked as disconnected\n", cl)
			}
		} else {
			cl.Stop()
			stopped = append(stopped, cl.Id)
		}
	}

	for _, s := range stopped {
		delete(c.Clients, s)
	}

	log.LogDebugf("Logger <%p> data sent to all [%d] clients...\n", c, len(c.Clients))
	log.LogDebugf("Logger <%p> data: %s\n", c, hex.EncodeToString(data))
}

func (c *ClientLogger) Add(cl *ClientSolarman) {
	c.lock.Lock()
	defer c.lock.Unlock()
	log.LogDebugf("Client [%p] registerd for Logger [%p]\n", cl, c)
	c.Clients[cl.Id] = cl
}

// Send will send data to the logger
//
// Extra operations can be performed too. Currently only logging
func (c *ClientLogger) Send(data []byte, from *ClientSolarman) {
	if !c.Running {
		return
	}
	log.LogDebugf("Logger <%p> sending data from <%p>\n", c, from)
	c.Conn.Write(data)
}

// DumpClients drops all ClientSolarman instances associated with the logger and returns them as a slice
func (c *ClientLogger) DumpClients() []*ClientSolarman {
	clients := make([]*ClientSolarman, 0)
	for _, cl := range c.Clients {
		clients = append(clients, cl)
	}
	c.Clients = nil
	return clients
}

// serialProbe send a predefined packet to the datalogger in order to acquire the serial number
func (c *ClientLogger) serialProbe() {
	probe := ReadHolding
	c.Conn.Write(probe.ToBytes())
}
