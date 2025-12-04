package providers

import (
	"net/http"
	"net/url"
)

type DeepseekProvider struct {
	BaseURL *url.URL
	APIKey  string
}

func NewDeepseekProvider(baseURL *url.URL, apiKey string) *DeepseekProvider {
	return &DeepseekProvider{
		BaseURL: baseURL,
		APIKey:  apiKey,
	}
}

func (p *DeepseekProvider) Director(req *http.Request) *http.Request {
	req.Host = p.BaseURL.Host
	req.Header.Set("Authorization", "Bearer "+p.APIKey)
	return req
}
