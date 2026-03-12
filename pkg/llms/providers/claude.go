package providers

import (
	"net/http"
)

type ClaudeProvider struct {
	BaseProvider
}

func (p *ClaudeProvider) Director(req *http.Request) *http.Request {
	req.Host = p.BaseURL.Host
	req.Header.Set("x-api-key", p.APIKey)
	return req
}
