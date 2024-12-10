package client

import (
	"encoding/hex"
	log "github.com/githubDante/go-solarman-proxy/logging"
	"net"
	"sync"
	"time"

	"github.com/githubDante/go-solarman-proxy/protocol"
)

const (
	writeTimeout = 200 * time.Millisecond // Deadline for socket write operations
)

type loggerBuffer struct {
	logger *ClientSolarman
	buf    []byte
}

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
	stoppedCh      chan *CommLogger
	Running        bool
	Id             uint32
	waitingForData bool
	dataBuffer     []*loggerBuffer
	bufferWanted   bool
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

		c.waitingForData = false
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

// EnableBuffering activates the logger write buffer. All messages will be sent sequentially
//
// The responses from the logger are still broadcasted to all clients
func (c *ClientLogger) EnableBuffering() {
	log.LogDebugf("Logger <%p> write buffer activated.\n", c)
	c.bufferWanted = true
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

	if c.pendingInBuffer() {
		buf := c.getFromBuffer()
		if buf != nil {
			c.Send(buf.buf, buf.logger)
		}
	}
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
	if c.waitingForData && c.bufferWanted {
		c.addToBuffer(data, from)
		return
	}
	log.LogDebugf("Logger <%p> sending data from <%p>\n", c, from)
	c.Conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	c.waitingForData = true
	_, err := c.Conn.Write(data)
	if err != nil {
		log.LogErrorf("Cannot communicate with logger <%p>\n", c)
		log.LogWarnf("Logger <%p> will be disconnected!\n", c)
		c.Stop()
	} else {
		if c.bufferWanted {
			log.LogInfof("Logger <%p> sending complete. Waiting for data [%t]\n", c, c.waitingForData)
		}
	}

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
	c.Conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	_, err := c.Conn.Write(probe.ToBytes())
	if err != nil {
		log.LogErrorf("SerialProbe failed, Cannot communicate with logger <%p>\n", c)
		log.LogWarnf("Logger <%p> will be disconnected!\n", c)
		c.Stop()
	}
}

func (c *ClientLogger) addToBuffer(buffer []byte, logger *ClientSolarman) {
	log.LogDebugf("Logger <%p> sending [%d bytes] to write buffer.\n", c, len(buffer))
	c.dataBuffer = append(c.dataBuffer, &loggerBuffer{logger: logger, buf: buffer})
}

func (c *ClientLogger) getFromBuffer() *loggerBuffer {
	if len(c.dataBuffer) == 0 {
		return nil
	}
	log.LogDebugf("Logger <%p> write buffer len [%d].\n", c, len(c.dataBuffer))
	top := c.dataBuffer[0]
	c.dataBuffer = c.dataBuffer[1:]
	log.LogDebugf("Logger <%p> got [%d bytes] message from buffer. Pending messages [%d].\n",
		c, len(top.buf), len(c.dataBuffer))
	return top
}

func (c *ClientLogger) pendingInBuffer() bool {
	return len(c.dataBuffer) > 0
}
