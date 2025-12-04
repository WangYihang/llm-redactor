package logging

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

func init() {
	zerolog.TimeFieldFormat = time.RFC3339Nano
}

type Loggers struct {
	System zerolog.Logger
	Data   zerolog.Logger
}

func New(logFile string) *Loggers {
	consoleWriter := zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: "15:04:05",
	}
	sysLog := zerolog.New(consoleWriter).
		With().
		Timestamp().
		Caller().
		Logger()

	fileWriter, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {

		sysLog.Fatal().Err(err).Str("path", logFile).Msg("cannot open data log file")
	}

	dataLog := zerolog.New(fileWriter).
		With().
		Timestamp().
		Logger()

	return &Loggers{
		System: sysLog,
		Data:   dataLog,
	}
}
