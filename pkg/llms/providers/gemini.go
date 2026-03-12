package providers

import (
	"net/http"
)

type GeminiProvider struct {
	BaseProvider
}

func (p *GeminiProvider) Director(req *http.Request) *http.Request {
	req.Host = p.BaseURL.Host
	// Gemini uses x-goog-api-key header or key= query parameter
	req.Header.Set("x-goog-api-key", p.APIKey)
	return req
}
