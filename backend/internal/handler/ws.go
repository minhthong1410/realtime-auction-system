package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"github.com/kurama/auction-system/backend/internal/app"
	"github.com/kurama/auction-system/backend/internal/logger"
	"github.com/kurama/auction-system/backend/internal/ws"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WSHandler struct {
	hub       *ws.Hub
	jwtSecret string
}

func NewWSHandler(ctx *app.Context) *WSHandler {
	h := &WSHandler{
		hub:       ctx.Hub,
		jwtSecret: ctx.Cfg.JWT.Secret,
	}

	ctx.Engine.GET("/ws", h.HandleWS)

	return h
}

func (h *WSHandler) HandleWS(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.Error("websocket upgrade failed", zap.Error(err))
		return
	}

	userID := ""
	if token := c.Query("token"); token != "" {
		userID = h.parseUserID(token)
	}

	client := ws.NewClient(h.hub, conn, userID)
	go client.WritePump()
	go client.ReadPump()
}

func (h *WSHandler) parseUserID(tokenStr string) string {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(h.jwtSecret), nil
	})
	if err != nil || !token.Valid {
		return ""
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return ""
	}

	userID, _ := claims["user_id"].(string)
	return userID
}
