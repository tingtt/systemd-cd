package logrus

import (
	"io"
	"systemd-cd/domain/model/logger"

	"github.com/sirupsen/logrus"
)

type Param struct {
	Level        *logrus.Level
	RepeatCaller *bool
	Output       io.Writer
	Formatter    logrus.Formatter
	BufferPool   logrus.BufferPool
}

type Level = logrus.Level

const (
	LevelPanic = logrus.PanicLevel
	LevelFatal = logrus.FatalLevel
	LevelError = logrus.ErrorLevel
	LevelWarn  = logrus.WarnLevel
	LevelInfo  = logrus.InfoLevel
	LevelDebug = logrus.DebugLevel
	LevelTrace = logrus.TraceLevel
)

func New(p Param) logger.LoggerI {
	logrus := logrus.New()

	if p.Level != nil {
		logrus.SetLevel(*p.Level)
	}
	if p.RepeatCaller != nil {
		logrus.SetReportCaller(*p.RepeatCaller)
	}
	if p.Output != nil {
		logrus.SetOutput(p.Output)
	}
	if p.Formatter != nil {
		logrus.SetFormatter(p.Formatter)
	}
	if p.BufferPool != nil {
		logrus.SetBufferPool(p.BufferPool)
	}

	return &WrapLogrus{Logger: logrus}
}

type WrapLogrus struct {
	*logrus.Logger
}

func (w *WrapLogrus) SetLevel(level logger.Level) error {
	// ログレベルを検証
	_, err := logrus.Level(level).MarshalText()
	if err != nil {
		// 不正なログレベル
		w.Error(err)
	} else {
		// ログレベルを適用
		w.Logger.SetLevel(logrus.Level(level))
		w.Infof("Log level: %q", w.Level.String())
	}

	return err
}
