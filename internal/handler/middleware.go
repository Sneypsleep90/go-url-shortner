package handler

import (
	"crypto/rand"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func RequestLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()
		requestID := rand.Text()
		c.Header("X-Request-ID", requestID)
		c.Next()

		status := c.Writer.Status()
		level := slog.LevelInfo
		switch {
		case status >= http.StatusInternalServerError:
			level = slog.LevelError
		case status >= http.StatusBadRequest:
			level = slog.LevelWarn
		}

		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}

		attrs := []slog.Attr{
			slog.String("request_id", requestID),
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.String("route", route),
			slog.String("remote_addr", c.Request.RemoteAddr),
			slog.Int("status", status),
			slog.Duration("duration", time.Since(started)),
		}
		requestErr := c.Errors.Last()
		switch {
		case requestErr != nil:
			attrs = append(attrs, slog.Any("error", requestErr.Err))
		case status >= http.StatusBadRequest:
			attrs = append(attrs, slog.String("error", http.StatusText(status)))
		}

		logger.LogAttrs(
			c.Request.Context(),
			level,
			"HTTP request",
			attrs...,
		)
	}
}
