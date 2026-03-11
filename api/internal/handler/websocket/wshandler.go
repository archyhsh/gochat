package websocket

import (
	"log"
	"net/http"

	wslogic "github.com/archyhsh/gochat/api/internal/logic/websocket"
	"github.com/archyhsh/gochat/api/internal/svc"
	gws "github.com/gorilla/websocket"
)

var upgrader = gws.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for this project
	},
}

func WsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Authenticate via token in query
		token := r.URL.Query().Get("token")
		if token == "" {
			http.Error(w, "Token missing", http.StatusUnauthorized)
			return
		}

		claims, err := svcCtx.JwtManager.ParseToken(token)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		userId := claims.UserID

		// 2. Upgrade to WebSocket
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("Upgrade error for user %d: %v", userId, err)
			return
		}

		// 3. Initialize logic and register connection
		l := wslogic.NewWsLogic(r.Context(), svcCtx)
		l.OnConnect(userId, conn)

		defer func() {
			l.OnDisconnect(userId, conn)
			conn.Close()
		}()

		// 4. Listen for client messages (heartbeats)
		for {
			mt, message, err := conn.ReadMessage()
			if err != nil {
				break
			}

			// Handle "ping" heartbeat to renew Redis lease
			if mt == gws.TextMessage && string(message) == "ping" {
				l.HandleHeartbeat(userId)
				_ = conn.WriteMessage(gws.TextMessage, []byte("pong"))
			}
		}
	}
}
