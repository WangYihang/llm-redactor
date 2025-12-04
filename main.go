package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/alecthomas/kong"
	"github.com/rs/zerolog"
	"github.com/wangyihang/llm-prism/pkg/llms/providers"
	"github.com/wangyihang/llm-prism/pkg/utils/logging"
	"github.com/wangyihang/llm-prism/pkg/utils/version"
)

type CLI struct {
	LogFile string `help:"The log file path." env:"LLM_PRISM_LOG_FILE" default:"llm-prism.jsonl"`
	Run     struct {
		ApiURL string `help:"The API base URL." env:"LLM_PRISM_API_URL" default:"https://api.deepseek.com/anthropic"`
		ApiKey string `help:"The API key." env:"LLM_PRISM_API_KEY" required:""`
		Host   string `help:"The host to listen on." env:"LLM_PRISM_HOST" default:"0.0.0.0"`
		Port   int    `help:"The port to listen on." env:"LLM_PRISM_PORT" default:"4000"`
	} `cmd:"" help:"Run the proxy server."`
	Version struct {
	} `cmd:"" help:"Print version information."`
}

func main() {
	var cli CLI
	ctx := kong.Parse(&cli,
		kong.Name("llm-prism"),
		kong.Description("A proxy server for LLM API requests."),
		kong.UsageOnError(),
	)

	logger := logging.GetLogger(cli.LogFile)

	switch ctx.Command() {
	case "run":
		logger.Info().
			Str("api_url", cli.Run.ApiURL).
			Str("host", cli.Run.Host).
			Int("port", cli.Run.Port).
			Str("log_file", cli.LogFile).
			Msg("starting proxy server")
		runProxy(logger, cli.Run.ApiURL, cli.Run.ApiKey, cli.Run.Host, cli.Run.Port)
	case "version":
		logger.Info().
			Str("version", version.GetVersionInfo().Version).
			Str("commit", version.GetVersionInfo().Commit).
			Str("date", version.GetVersionInfo().Date).
			Msg("version information")
		fmt.Println(version.GetVersionInfo().JSON())
	}
}

func runProxy(logger zerolog.Logger, apiURL, apiKey, host string, port int) {
	baseURL, err := url.Parse(apiURL)
	if err != nil {
		logger.Error().Err(err).Msg("failed to parse API URL")
		return
	}
	deepseekProvider := providers.NewDeepseekProvider(baseURL, apiKey)
	proxy := httputil.NewSingleHostReverseProxy(deepseekProvider.BaseURL)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		deepseekProvider.Director(req)
	}
	proxy.ModifyResponse = func(resp *http.Response) error {
		return nil
	}
	addr := fmt.Sprintf("%s:%d", host, port)
	logger.Info().Str("address", addr).Msg("proxy server started")
	err = http.ListenAndServe(addr, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxy.ServeHTTP(w, r)
	}))
	if err != nil {
		logger.Error().Err(err).Msg("failed to start proxy server")
		return
	}
}
