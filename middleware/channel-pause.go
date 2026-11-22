package middleware

import (
	"net/http"

	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// ChannelPause blocks new relay requests while preserving every channel's
// individual enabled/disabled state. Existing task fetches remain available.
func ChannelPause() gin.HandlerFunc {
	return func(c *gin.Context) {
		enabled, message := model.GetChannelPauseState()
		if enabled && c.Request.Method != http.MethodGet {
			if message == "" {
				message = "Channel service is temporarily unavailable"
			}
			abortWithOpenAiMessage(c, http.StatusServiceUnavailable, message)
			return
		}
		c.Next()
	}
}
