package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/go-playground/validator/v10"
)

func requestErrorMessage(err error) string {
	if errors.Is(err, io.EOF) {
		return "request body is empty"
	}

	switch err := err.(type) {
	case validator.ValidationErrors:
		switch err[0].Tag() {
		case "required":
			return "original_url is required"
		case "excludes":
			return "original_url must not contain spaces"
		case "http_url":
			return "original_url must be  valid HTTP or HTTPS URL"
		default:
			return "original_url is invalid"
		}
	case *json.UnmarshalTypeError:
		if err.Field == "original_url" {
			return "original_url must be string"
		}
		return "request body must be  JSON object"
	case *http.MaxBytesError:
		return fmt.Sprintf("request body exceeds %d bytes", err.Limit)
	default:
		return "request body must be valid JSON"
	}
}
