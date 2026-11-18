package handler

import "github.com/gin-gonic/gin"

func (h *Handler) RegisterRoutes(router gin.IRouter) {
	router.POST("/links", h.CreateCode)
	router.GET("/:code", h.GetUrlByCode)
}
