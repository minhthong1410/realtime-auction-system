package middleware

import "github.com/gin-gonic/gin"

func I18nMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		lang := c.DefaultQuery("lang", c.GetHeader("Accept-Language"))
		if lang == "" {
			lang = "en"
		}
		c.Set("lang", lang)
		c.Next()
	}
}
