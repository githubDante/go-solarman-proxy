package server

import (
	"encoding/hex"
	"errors"
	"fmt"
	log "github.com/githubDante/go-solarman-proxy/logging"
	"net"
	"sync"
	"time"

	"github.com/githubDante/go-solarman-proxy/client"
)

type V5ProxyServer struct {
	Host        string
	ClientsPort uint16
	LoggersPort uint16

	loggersL net.Listener
	clientsL net.Listener

	// Data-loggers connected to the proxy
	//  map[ClientLogger.Serial]*client.ClientLogger
	loggers map[uint32]*client.ClientLogger
	// Data-loggers with unknown serial
	//  map[ClientLogger.Id]*client.ClientLogger
	martians map[uint32]*client.ClientLogger
	// Clients for which the serial number is unknown or the logger of which was disconnected
	//  map[ClientSolarman.Id]*ClientSolarman
	pending map[uint32]*client.ClientSolarman

	// Data loggers serial numbers receiver
	loggersComm chan *client.CommLogger
	// Notification channel for stopped/disconnected data-loggers
	loggerStopped chan *client.CommLogger
	// Clients serial numbers receiver
	clientsComm chan *client.CommSolarman
	//
	broadcastComm chan []byte

	blocker sync.Mutex
	mapSync sync.Mutex
}

func NewProxy(host string, loggersPort int) *V5ProxyServer {
	return &V5ProxyServer{
		Host:        host,
		ClientsPort: 8899, //  Same as the data-logger TCP server
		LoggersPort: uint16(loggersPort),

		loggersComm:   make(chan *client.CommLogger),
		clientsComm:   make(chan *client.CommSolarman),
		loggerStopped: make(chan *client.CommLogger),
		broadcastComm: make(chan []byte),

		loggers:  make(map[uint32]*client.ClientLogger),
		martians: make(map[uint32]*client.ClientLogger),
		pending:  make(map[uint32]*client.ClientSolarman),

		blocker: sync.Mutex{},
	}
}

// Wait - will block the calling function while the server is running i.e. forever
func (s *V5ProxyServer) Wait() {
	s.blocker.Lock()
}

// Serve - Creates listeners and starts the proxy loops
func (s *V5ProxyServer) Serve() error {

	var err error
	s.clientsL, err = net.Listen("tcp4", fmt.Sprintf("%s:%d", "0.0.0.0", s.ClientsPort))
	if err != nil {
		return errors.New("cannot create clients listener: " + err.Error())
	}
	s.loggersL, err = net.Listen("tcp4", fmt.Sprintf("%s:%d", s.Host, s.LoggersPort))

	if err != nil {
		return errors.New("cannot create loggers listener: " + err.Error())
	}

	log.LogInfof("[Proxy] sockets created. ClientPort [%d] - LoggersPort [%d]\n", 8899, s.LoggersPort)
	go s.loggersConn()
	go s.clientsConn()
	go s.handleBroadcasts()
	go s.janitor()

	s.blocker.Lock()
	return nil
}

// loggersConn Connection manager for data logger connections
func (s *V5ProxyServer) loggersConn() {

	log.LogInfof("[Loggers-Proxy] waiting for logger connections\n")

	go func() {
		for {
			select {
			case logger := <-s.loggersComm:
				s.loggers[logger.Serial] = logger.Logger
				log.LogInfof("Logger <%s> provided serial [%d]\n",
					logger.Logger.Conn.RemoteAddr().String(), logger.Serial)
				s.checkPending(logger)
			case logger := <-s.loggerStopped:
				s.handleLoggerDisconnect(logger)
			}
		}
	}()

	for {
		conn, err := s.loggersL.Accept()
		log.LogInfof("New loggger connection from: %s\n", conn.RemoteAddr().String())
		if err != nil {
			log.LogErrorf("Logger connection error: %s\n", err.Error())
			continue
		}
		cl := client.NewLoggerClient(conn, s.loggersComm, s.loggerStopped)
		s.martians[cl.Id] = cl

		go cl.Run()
	}
}

// checkPending - check for any clients not associated with a freshly connected data-logger
//
// When such client is found bindings between the client and the logger are created
func (s *V5ProxyServer) checkPending(logger *client.CommLogger) {
	s.mapSync.Lock()
	defer s.mapSync.Unlock()

	assigned := make([]uint32, 0)
	for _, cl := range s.pending {
		if cl.Serial == logger.Serial {
			s.loggers[logger.Serial].Add(cl)
			cl.AddLogger(logger.Logger)
			assigned = append(assigned, cl.Id)
		}
	}
	for _, as := range assigned {
		delete(s.pending, as)
	}
}

func (s *V5ProxyServer) clientsConn() {

	log.LogInfof("[Clients-Proxy] waiting for client connections\n")
	go s.manageClients()
	for {
		conn, err := s.clientsL.Accept()
		if err != nil {
			log.LogErrorf("Client connection error: %s\n", err.Error())
			continue
		}
		cl := client.NewSolarmanClient(conn, s.clientsComm, s.broadcastComm)
		log.LogInfof("New solarman client [%s] connected\n", conn.RemoteAddr().String())

		s.mapSync.Lock()
		s.pending[cl.Id] = cl
		s.mapSync.Unlock()
		go cl.Run()
	}
}

// manageClients Clients will be assigned to the logger (if available)
//
// If logger with serial number requested by the client is already connected o the proxy
// then the client will be associated with it, otherwise the client will be disconnected.
func (s *V5ProxyServer) manageClients() {
	for {
		cl := <-s.clientsComm // Serial received from a solarman client
		logger, ok := s.loggers[cl.Serial]
		if ok && logger.Running {
			logger.Add(cl.Client)
			cl.Client.Logger = logger
		} else {
			log.LogWarnf("No logger connected for [%d]\n", cl.Serial)
		}
	}
}

func (s *V5ProxyServer) handleBroadcasts() {
	for {

		data := <-s.broadcastComm
		log.LogInfof("Server - broadcasting: %s\n", hex.EncodeToString(data))
		for _, logger := range s.martians {
			if logger.Serial == 0 {
				log.LogInfof("Server - broadcasting to %p\n", logger)
				logger.Conn.Write(data)
			}
		}
	}
}

func (s *V5ProxyServer) janitor() {
	ticker := time.NewTicker(30 * time.Second)
	for {
		<-ticker.C
		s.checkRunningLoggers()
		s.checkPendingClients()
	}
}

// checkRunningLoggers check the state of all connected data-loggers
//
// Clients from disconnected loggers are transferred back to the pending structure
// Not running martians are discarded
func (s *V5ProxyServer) checkRunningLoggers() {
	s.mapSync.Lock()
	defer s.mapSync.Unlock()

	notRunning := make([]uint32, 0)
	for _, logger := range s.loggers {
		if !logger.Running {
			clients := logger.DumpClients()
			log.LogDebugf("[Server] Logger <%p> not runnig. Dumped [%d] clients.\n", logger, len(clients))
			logger.Stop()
			for _, cl := range clients {
				cl.Logger = nil
				s.pending[cl.Id] = cl
			}
			notRunning = append(notRunning, logger.Serial)
		}
	}
	for _, lId := range notRunning {
		delete(s.loggers, lId)
	}
	mCleanup := make([]uint32, 0)
	for _, m := range s.martians {
		if !m.Running {
			m.Stop()
			mCleanup = append(mCleanup, m.Id)
		} else if m.Serial != 0 {
			mCleanup = append(mCleanup, m.Id)
		}
	}
	for _, lId := range mCleanup {
		delete(s.martians, lId)
	}
	log.LogDebugf("[Server] loggers: known [%d] - unknown [%d]\n", len(s.loggers), len(s.martians))
}

// handleLoggerDisconnect transfers the clients currently associated with the data-logger
// to the pending list and removes the logger association with the proxy
func (s *V5ProxyServer) handleLoggerDisconnect(logger *client.CommLogger) {
	s.mapSync.Lock()
	defer s.mapSync.Unlock()
	clients := logger.Logger.DumpClients()
	for _, cl := range clients {
		s.pending[cl.Id] = cl
	}
	logger.Logger.Clients = nil
	logger.Logger.Stop()
	delete(s.loggers, logger.Serial)
	log.LogDebugf("[Server] Logger <%p> disconnected. Active loggers [%d]\n", logger, len(s.loggers))
}

// checkPendingClients iterates over the pending solarman clients and deletes the disconnected ones
func (s *V5ProxyServer) checkPendingClients() {
	s.mapSync.Lock()
	defer s.mapSync.Unlock()

	notRunning := make([]uint32, 0)
	for _, cl := range s.pending {
		if !cl.Running {
			notRunning = append(notRunning, cl.Id)
		}
	}
	for _, nId := range notRunning {
		delete(s.pending, nId)
	}
	log.LogDebugf("[Server] pending clients [%d]\n", len(s.pending))
}
