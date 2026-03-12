package providers

import (
	"net/http"
	"net/url"
)

type Provider interface {
	Director(req *http.Request) *http.Request
}

type BaseProvider struct {
	BaseURL *url.URL
	APIKey  string
}

func NewBaseProvider(baseURL *url.URL, apiKey string) Provider {
	return &BaseProvider{BaseURL: baseURL, APIKey: apiKey}
}

func (p *BaseProvider) Director(req *http.Request) *http.Request {
	req.Host = p.BaseURL.Host
	req.Header.Set("Authorization", "Bearer "+p.APIKey)
	return req
}

func GetProvider(name string, baseURL *url.URL, apiKey string) Provider {
	switch name {
	case "deepseek":
		return &DeepseekProvider{
			BaseURL: baseURL,
			APIKey:  apiKey,
		}
	case "kimi":
		return &KimiProvider{
			BaseURL: baseURL,
			APIKey:  apiKey,
		}
	case "claude":
		return &ClaudeProvider{
			BaseProvider: BaseProvider{
				BaseURL: baseURL,
				APIKey:  apiKey,
			},
		}
	case "gemini":
		return &GeminiProvider{
			BaseProvider: BaseProvider{
				BaseURL: baseURL,
				APIKey:  apiKey,
			},
		}
	case "openai":
		return &BaseProvider{
			BaseURL: baseURL,
			APIKey:  apiKey,
		}
	default:
		return NewBaseProvider(baseURL, apiKey)
	}
}
