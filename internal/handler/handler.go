package handler

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/Sneypsleep90/go-url-shortner/internal/domain"
	"github.com/gin-gonic/gin"
)

const maxRequestBodySize = 1 << 20

type Handler struct {
	service LinkService
}

func NewHandler(service LinkService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) CreateCode(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxRequestBodySize)

	var request createLinkRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		message := requestErrorMessage(err)
		c.Error(fmt.Errorf("%s: %w", message, err))
		c.JSON(http.StatusBadRequest, gin.H{"error": message})
		return
	}

	link, err := h.service.CreateCode(c.Request.Context(), domain.Link{OriginalURL: request.OriginalURL})
	if err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusCreated, linkResponse{
		ShortCode:   link.ShortCode,
		OriginalURL: link.OriginalURL,
	})
}

func (h *Handler) GetUrlByCode(c *gin.Context) {
	link, err := h.service.GetUrlByCode(c.Request.Context(), c.Param("code"))
	if errors.Is(err, domain.ErrNotFound) {
		c.Error(err)
		c.JSON(http.StatusNotFound, gin.H{"error": "link not found"})
		return
	}
	if err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, linkResponse{
		ShortCode:   link.ShortCode,
		OriginalURL: link.OriginalURL,
	})
}
