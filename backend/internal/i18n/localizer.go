package i18n

import (
	"github.com/gin-gonic/gin"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// T translates a message ID using the language from the gin context.
func T(c *gin.Context, msgID string, data map[string]interface{}) string {
	lang := c.GetString("lang")
	localizer := i18n.NewLocalizer(Bundle, lang)

	msg, err := localizer.Localize(&i18n.LocalizeConfig{
		MessageID:    msgID,
		TemplateData: data,
	})
	if err != nil {
		return msgID
	}
	return msg
}
