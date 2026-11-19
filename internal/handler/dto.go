package handler

type createLinkRequest struct {
	OriginalURL string `json:"original_url" binding:"required,excludes= ,http_url"`
}

type linkResponse struct {
	ShortCode   string `json:"short_code"`
	OriginalURL string `json:"original_url"`
}
