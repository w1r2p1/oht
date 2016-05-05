package p2p

import (
	"crypto/ecdsa"
	"fmt"
	"log"
	"net/http"
	"time"

	"../types"
)

type EventFunc func(Manager *Manager, Peer *Peer)

type Manager struct {
	PrivateKey      *ecdsa.PrivateKey
	Server          *Server
	MaxPeers        int
	MaxPendingPeers int
	Peers           map[*Peer]bool
	Broadcast       chan *types.Message
	Receive         chan *types.Message
	Register        chan *Peer
	Unregister      chan *Peer
	OnConnect       EventFunc
	OnClose         EventFunc
	lastLookup      time.Time
}

func InitializeNetworkManager() *Manager {
	return &Manager{
		// Need to add a new message with custom type to send out
		// Info about current node, like onion address
		Server:          &Server{},
		MaxPeers:        8,
		MaxPendingPeers: 8,
		Broadcast:       make(chan *types.Message, maxMessageSize),
		Receive:         make(chan *types.Message, maxMessageSize),
		Register:        make(chan *Peer, maxMessageSize),
		Unregister:      make(chan *Peer, maxMessageSize),
		Peers:           make(map[*Peer]bool, maxMessageSize),
		OnConnect:       nil,
		OnClose:         nil,
		lastLookup:      time.Now(),
	}
}

func (Manager *Manager) Start(port int) {
	Manager.Server.Start(port)
	for {
		select {
		case p := <-Manager.Register:
			Manager.Peers[p] = true
			log.Println("Peer connection established: ", p.OnionHost)
			fmt.Printf("oht> ")
			if Manager.OnConnect != nil {
				go Manager.OnConnect(Manager, p)
			}
		case p := <-Manager.Unregister:
			if _, ok := Manager.Peers[p]; ok {
				delete(Manager.Peers, p)
				close(p.Send)
				if Manager.OnClose != nil {
					go Manager.OnClose(Manager, p)
				}
				p.Websocket.Close()
			}
		case m := <-Manager.Broadcast:
			for p := range Manager.Peers {
				select {
				case p.Send <- m:
				default:
					close(p.Send)
					delete(Manager.Peers, p)
				}
			}
		case m := <-Manager.Receive:
			log.Println("")
			log.Println("[", m.Timestamp, "] ", m.Username, " : ", m.Body)
			fmt.Printf("oht> ")
		}
	}
}

func (Manager *Manager) Stop() {
}

func (Manager *Manager) DumpPeers() {
	for p := range Manager.Peers {
		log.Println("Connection")
		log.Println("Connection: ", p.OnionHost)
	}
}

// Serve handles websocket requests from the peer
func (Manager *Manager) Serve(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	p := &Peer{Send: make(chan *types.Message, 256), Websocket: ws}
	Manager.Register <- p
	go p.writeMessages()
	p.readMessages()
}