package logger

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"sync"
	"time"
)

type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
)

var (
	mu       sync.Mutex
	minLevel = INFO
	colors   bool
	writer   *log.Logger
	initOnce sync.Once
)

var levelColors = map[Level]string{
	DEBUG: "\033[36m",
	INFO:  "\033[32m",
	WARN:  "\033[33m",
	ERROR: "\033[31m",
}

var levelPrefixes = map[Level]string{
	DEBUG: "[DEBUG]",
	INFO:  "[INFO] ",
	WARN:  "[WARN] ",
	ERROR: "[ERROR]",
}

const reset = "\033[0m"

func init() {
	initOnce.Do(func() {
		writer = log.New(os.Stderr, "", 0)
		if !isCI() {
			colors = true
		}
	})
}

func isCI() bool {
	if os.Getenv("CI") != "" {
		return true
	}
	if os.Getenv("TERM") == "" {
		return true
	}
	return false
}

func SetLevel(level Level) {
	mu.Lock()
	minLevel = level
	mu.Unlock()
}

func SetColors(enabled bool) {
	mu.Lock()
	colors = enabled
	mu.Unlock()
}

func Debug(format string, v ...interface{}) {
	logAt(DEBUG, format, v...)
}

func Info(format string, v ...interface{}) {
	logAt(INFO, format, v...)
}

func Warn(format string, v ...interface{}) {
	logAt(WARN, format, v...)
}

func Error(format string, v ...interface{}) {
	logAt(ERROR, format, v...)
}

func logAt(level Level, format string, v ...interface{}) {
	mu.Lock()
	if level < minLevel {
		mu.Unlock()
		return
	}
	enableColors := colors
	mu.Unlock()

	msg := fmt.Sprintf(format, v...)
	timestamp := time.Now().Format("2006/01/02 15:04:05")

	mu.Lock()
	prefix := levelPrefixes[level]
	mu.Unlock()

	if enableColors {
		color := levelColors[level]
		writer.Print(fmt.Sprintf("%s%s%s%s %s", timestamp, color, prefix, reset, msg))
	} else {
		writer.Print(fmt.Sprintf("%s %s %s", timestamp, prefix, msg))
	}
}

func WithCallers(depth int) string {
	_, file, line, _ := runtime.Caller(depth)
	return fmt.Sprintf("%s:%d", file, line)
}
