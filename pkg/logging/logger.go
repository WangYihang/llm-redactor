package logging

import (
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

var (
	loggerOnce    sync.Once
	globalLogger  zerolog.Logger
	loggerInitErr error
)

func init() {
	zerolog.TimeFieldFormat = time.RFC3339Nano
}

// GetLogger lazily instantiates a shared zerolog logger instance.
func GetLogger(logFile string) zerolog.Logger {
	loggerOnce.Do(func() {
		fileWriter, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			loggerInitErr = err
			return
		}
		consoleWriter := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: "2006-01-02 15:04:05.000"}
		multi := zerolog.MultiLevelWriter(consoleWriter, fileWriter)
		globalLogger = zerolog.New(multi).With().Timestamp().Caller().Logger()
	})
	if loggerInitErr != nil {
		panic(loggerInitErr)
	}
	return globalLogger
}
