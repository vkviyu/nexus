package logutil

import (
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/sirupsen/logrus"
)

var DefaultLogDir = "logs"
var DefaultRotateTime = 24 * time.Hour
var DefaultMaxAge = 30 * 24 * time.Hour
var DefaultFormatter = &logrus.JSONFormatter{
	TimestampFormat: "2006-01-02 15:04:05",
}

var DefaultLogLevel = logrus.InfoLevel

type RotateLogger struct {
	*logrus.Logger
}

type RotateLoggerConfig struct {
	LogDir     string
	RotateTime time.Duration
	MaxAge     time.Duration
	Formatter  logrus.Formatter
	LogLevel   logrus.Level
}

func NewRotateLogger(config *RotateLoggerConfig) (*RotateLogger, error) {
	if config == nil {
		config = &RotateLoggerConfig{
			LogDir:     DefaultLogDir,
			RotateTime: DefaultRotateTime,
			MaxAge:     DefaultMaxAge,
		}
	}
	if config.LogDir == "" {
		config.LogDir = DefaultLogDir
	}
	if config.RotateTime == 0 {
		config.RotateTime = 24 * time.Hour
	}
	if config.MaxAge == 0 {
		config.MaxAge = 30 * 24 * time.Hour
	}
	writer, err := rotatelogs.New(
		config.LogDir+"/%Y%m%d.log",
		rotatelogs.WithLinkName(config.LogDir+"/latest.log"),
		rotatelogs.WithRotationTime(config.RotateTime),
		rotatelogs.WithMaxAge(config.MaxAge),
	)
	if err != nil {
		return nil, err
	}
	logger := logrus.New()
	logger.SetOutput(writer)
	if config.Formatter == nil {
		config.Formatter = DefaultFormatter
	}
	logger.SetFormatter(config.Formatter)
	if config.LogLevel == 0 {
		config.LogLevel = DefaultLogLevel
	}
	logger.SetLevel(config.LogLevel)
	return &RotateLogger{logger}, nil
}
