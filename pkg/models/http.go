package models

import (
	"io"
	"log/slog"
	"net/http"
)

type HTTPHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type HTTPRequest struct {
	Method  string       `json:"method"`
	URL     string       `json:"url"`
	Host    string       `json:"host"`
	Headers []HTTPHeader `json:"headers"`
	Body    interface{}  `json:"body"`
}

type HTTPResponse struct {
	StatusCode int          `json:"status_code"`
	Status     string       `json:"status"`
	Headers    []HTTPHeader `json:"headers"`
	Body       interface{}  `json:"body"`
}

func NewHTTPRequest(req *http.Request) *HTTPRequest {
	headers := make([]HTTPHeader, 0, len(req.Header))
	for name, values := range req.Header {
		for _, value := range values {
			headers = append(headers, HTTPHeader{Name: name, Value: value})
		}
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		slog.Error("failed to read request body", "error", err)
		return nil
	}
	return &HTTPRequest{
		Method:  req.Method,
		URL:     req.URL.String(),
		Host:    req.Host,
		Headers: headers,
		Body:    string(body),
	}
}

func NewHTTPResponse(resp *http.Response) *HTTPResponse {
	headers := make([]HTTPHeader, 0, len(resp.Header))
	for name, values := range resp.Header {
		for _, value := range values {
			headers = append(headers, HTTPHeader{Name: name, Value: value})
		}
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("failed to read response body", "error", err)
		return nil
	}
	return &HTTPResponse{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Headers:    headers,
		Body:       string(body),
	}
}
