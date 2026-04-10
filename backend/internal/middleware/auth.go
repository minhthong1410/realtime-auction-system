package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/kurama/auction-system/backend/internal/httputil"
)

func AuthMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			httputil.RenderGinJSON(http.StatusUnauthorized, c, httputil.NewErrorResponse(c, http.StatusUnauthorized, "missing authorization header"))
			c.Abort()
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			httputil.RenderGinJSON(http.StatusUnauthorized, c, httputil.NewErrorResponse(c, http.StatusUnauthorized, "invalid authorization format"))
			c.Abort()
			return
		}

		token, err := jwt.Parse(parts[1], func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			httputil.RenderGinJSON(http.StatusUnauthorized, c, httputil.NewErrorResponse(c, http.StatusUnauthorized, "invalid or expired token"))
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			httputil.RenderGinJSON(http.StatusUnauthorized, c, httputil.NewErrorResponse(c, http.StatusUnauthorized, "invalid token claims"))
			c.Abort()
			return
		}

		userID, ok := claims["user_id"].(string)
		if !ok {
			httputil.RenderGinJSON(http.StatusUnauthorized, c, httputil.NewErrorResponse(c, http.StatusUnauthorized, "invalid user_id in token"))
			c.Abort()
			return
		}

		username, _ := claims["username"].(string)

		// Set request context
		reqCtx := &httputil.RequestContext{
			UserID:   userID,
			Username: username,
		}
		c.Request = httputil.WithRequestContext(c.Request, reqCtx)
		c.Next()
	}
}
