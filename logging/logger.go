package logging

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	logger    *logrus.Logger
	logChan   chan *logrus.Entry
	once      sync.Once
	logBuffer sync.Pool
)

const (
	logQueueSize = 10000
)

func Init() {
	once.Do(func() {
		logger = logrus.New()
		logger.SetLevel(logrus.InfoLevel)

		// Configure lumberjack for log rotation
		logRotator := &lumberjack.Logger{
			Filename:   "app.log",
			MaxSize:    50, // megabytes
			MaxBackups: 3,
			MaxAge:     7, //days
			Compress:   true,
		}
		// Set multi-writer to log to both file and standard output
		mw := io.MultiWriter(os.Stdout, logRotator)
		logger.SetOutput(mw)

		// Use a text formatter
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
			CallerPrettyfier: func(f *runtime.Frame) (string, string) {
				filename := strings.Split(f.File, "/")
				return fmt.Sprintf("%s()", f.Function), fmt.Sprintf("%s:%d", filename[len(filename)-1], f.Line)
			},
		})

		logger.SetReportCaller(true)

		logChan = make(chan *logrus.Entry, logQueueSize)
		logBuffer = sync.Pool{
			New: func() interface{} {
				return new(logrus.Entry)
			},
		}

		go consumeLogs()
	})
}

func consumeLogs() {
	for entry := range logChan {
		entry.Logger.WithFields(entry.Data).Log(entry.Level, entry.Message)
		logBuffer.Put(entry)
	}
}

func Log(level logrus.Level, message string, fields logrus.Fields) {
	if logger == nil {
		// Fallback to standard logger if not initialized
		fmt.Printf("logger not initialized: %s\n", message)
		return
	}

	entry := logBuffer.Get().(*logrus.Entry)
	entry.Logger = logger
	entry.Level = level
	entry.Message = message
	entry.Time = time.Now()
	entry.Data = fields

	select {
	case logChan <- entry:
	default:
		// Drop the log if the channel is full
		// In a real-world scenario, you might want to increment a metric here
	}
}

func Info(message string, fields logrus.Fields) {
	Log(logrus.InfoLevel, message, fields)
}

func Warn(message string, fields logrus.Fields) {
	Log(logrus.WarnLevel, message, fields)
}

func Error(message string, fields logrus.Fields) {
	Log(logrus.ErrorLevel, message, fields)
}
