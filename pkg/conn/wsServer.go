package conn

import (
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/miekg/dns"
	"log"
	"net/http"
	"sync"
)

// WsServer is websocket server
type WsServer struct {
	clients     map[*websocket.Conn]bool
	upgrader    websocket.Upgrader
	receiveChan chan *dns.Msg
	sync.RWMutex
}

// NewWsServer generates websocket server
func NewWsServer() WsServer {
	return WsServer{
		clients: make(map[*websocket.Conn]bool),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		receiveChan: make(chan *dns.Msg),
	}
}

func (wsSrv *WsServer) ReceiveChan() <-chan *dns.Msg {
	return wsSrv.receiveChan
}

// webSocketHandler is websocket connection handler
func (wsSrv *WsServer) webSocketHandler(ctx *gin.Context) {
	ws, err := wsSrv.upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		log.Printf("[ERROR] conn: Failed to upgrade websocket: %v", err)
		ctx.String(http.StatusBadRequest, "Can't upgrade websocket\n", err)
		return
	}

	defer ws.Close()

	wsSrv.clients[ws] = true

	for {
		m := new(dns.Msg)

		err := ws.ReadJSON(m)
		if err != nil {
			log.Printf("[WARN] conn: Failed to read JSON: %v", err)
			break
		}

		wsSrv.receiveChan <- m
	}
}

// SendMessage sends message with websocket
func (wsSrv *WsServer) SendMessage(msg *dns.Msg) {
	log.Printf("[INFO] conn: Send dns message: %v", msg) // For debug

	// Broadcast
	for client := range wsSrv.clients {
		err := client.WriteJSON(msg)
		if err != nil {
			log.Printf("[WARN] conn: Failed to send JSON: %v", err)
			client.Close()
			delete(wsSrv.clients, client)
		}
	}
}
