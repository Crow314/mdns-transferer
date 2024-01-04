package conn

import (
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
)

var WsSrv WsServer

type Server struct {
	Engine *gin.Engine
}

// Up starts server
func (server *Server) Up() {
	server.Engine = gin.Default()
	WsSrv = NewWsServer()

	server.setRoutes()

	err := server.Engine.Run(":15353")
	if err != nil {
		log.Fatalf("[FATAL] httpd: Failed to start http server")
		return
	}
}

func (server *Server) setRoutes() {
	server.Engine.GET("/ws", webSocketConnector)

	server.Engine.GET("/ping", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{
			"status": "OK",
		})
	})
}

func webSocketConnector(ctx *gin.Context) {
	WsSrv.webSocketHandler(ctx)
}
