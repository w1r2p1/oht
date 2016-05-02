package network

import (
	"github.com/gin-gonic/gin"
	"strconv"
	"sync"
	"time"
)

const (
	defaultConnectionTimeout = 15 * time.Second
	refreshPeersInterval     = 90 * time.Second
	maxConnections           = 8
)

type Server struct {
	Websocket *gin.Engine
	lock      sync.Mutex
	Port      int
}

func (server *Server) Start(port int) {
	server.lock.Lock()
	defer server.lock.Unlock()
	server.Port = port
	gin.SetMode(gin.ReleaseMode)
	server.Websocket = gin.Default()
	server.Websocket.GET("/ws", func(c *gin.Context) {
		Manager.Serve(c.Writer, c.Request)
	})
	go server.Websocket.Run("127.0.0.1:" + strconv.Itoa(server.Port))
}

func (server *Server) PeerCount() int {
	return 0
}
