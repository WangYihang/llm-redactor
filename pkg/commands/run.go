package commands

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/wangyihang/llm-prism/pkg/config"
	"github.com/wangyihang/llm-prism/pkg/proxy"
	"github.com/wangyihang/llm-prism/pkg/redactor"
	"github.com/wangyihang/llm-prism/pkg/utils/logging"
)

func Run(cli *config.CLI, logs *logging.Loggers) {
	rdr, err := redactor.New(cli.RedactorRules, logs.Detection)
	if err != nil {
		logs.System.Warn().Err(err).Msg("failed to load redactor rules, skipping redaction")
	}

	rp, err := proxy.Setup(cli, rdr, logs)
	if err != nil {
		logs.System.Fatal().Err(err).Msg("failed to setup reverse proxy")
	}

	addr := fmt.Sprintf("%s:%d", cli.Run.Host, cli.Run.Port)
	logs.System.Info().Str("addr", addr).Msg("started")

	err = http.ListenAndServe(addr, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t := time.Now()
		requestID := uuid.New().String()
		rb := new(bytes.Buffer)
		if r.Body != nil {
			r.Body = io.NopCloser(io.TeeReader(r.Body, rb))
		}
		sw := &proxy.Spy{ResponseWriter: w, Buf: new(bytes.Buffer), Code: http.StatusOK}

		// Inject request ID into context for downstream components
		r = r.WithContext(proxy.WithRequestID(r.Context(), requestID))

		rp.ServeHTTP(sw, r)

		reqEvt := zerolog.Dict().Str("id", requestID).Str("method", r.Method).Str("path", r.URL.Path)
		proxy.EnrichLogEvent(reqEvt, rb.Bytes(), r.Header, logs.System)

		resEvt := zerolog.Dict().Int("status", sw.Code)
		proxy.EnrichLogEvent(resEvt, sw.Buf.Bytes(), sw.Header(), logs.System)

		logs.Data.Info().
			Str("id", requestID).
			Dur("duration", time.Since(t)).
			Dict("http", zerolog.Dict().Dict("request", reqEvt).Dict("response", resEvt)).
			Msg("")
	}))

	if err != nil {
		logs.System.Fatal().Err(err).Msg("failed")
	}
}
