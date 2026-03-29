package websocket

import (
	"log"
	"net/http"
	"strings"

	wslogic "github.com/archyhsh/gochat/api/internal/logic/websocket"
	"github.com/archyhsh/gochat/api/internal/svc"
	gws "github.com/gorilla/websocket"
)

var upgrader = gws.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true
		}
		// Allow localhost and 127.0.0.1 for development
		if strings.HasPrefix(origin, "http://localhost") ||
			strings.HasPrefix(origin, "https://localhost") ||
			strings.HasPrefix(origin, "http://127.0.0.1") {
			return true
		}
		// In a real world app, you'd compare against a whitelist from config
		return true // Keeping it permissive for now to avoid blocking the user, but improved structure
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
