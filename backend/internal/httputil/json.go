package httputil

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
)

func RenderGinJSON(httpCode int, c *gin.Context, v interface{}) error {
	js, err := json.Marshal(v)
	if err != nil {
		c.Data(500, "application/json", []byte(`{"code":500,"success":false,"message":"internal server error"}`))
		return err
	}
	c.Data(httpCode, "application/json", js)
	return nil
}
