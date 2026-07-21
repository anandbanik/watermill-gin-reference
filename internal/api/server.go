// Package api exposes the HTTP transport built on Gin. The POST /orders
// endpoint decodes the JSON body and delegates to the shared order.Service —
// the same method the Kafka listener calls.
package api

import (
	"net/http"

	"github.com/anandbanik/watermill-gin/internal/order"
	"github.com/gin-gonic/gin"
)

// NewRouter wires the HTTP routes to the shared order service.
func NewRouter(svc *order.Service) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.POST("/orders", func(c *gin.Context) {
		var o order.Order
		if err := c.ShouldBindJSON(&o); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Same handler the Kafka listener invokes.
		res, err := svc.Process(c.Request.Context(), "api", o)
		if err != nil {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, res)
	})

	return r
}
