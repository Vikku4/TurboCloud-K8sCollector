package log

import (
	"log"
	"os"
)

type Logger struct {
	*log.Logger
	level Level
}

type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

func ParseLevel(s string) Level {
	switch s {
	case "debug":
		return LevelDebug
	case "warn":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelInfo
	}
}

func New(levelStr string) *Logger {
	return &Logger{
		Logger: log.New(os.Stdout, "", log.LstdFlags|log.LUTC|log.Lshortfile),
		level:  ParseLevel(levelStr),
	}
}

func (l *Logger) Debugf(format string, v ...any) {
	if l.level <= LevelDebug {
		l.Printf("[DEBUG] "+format, v...)
	}
}

func (l *Logger) Infof(format string, v ...any) {
	if l.level <= LevelInfo {
		l.Printf("[INFO] "+format, v...)
	}
}

func (l *Logger) Warnf(format string, v ...any) {
	if l.level <= LevelWarn {
		l.Printf("[WARN] "+format, v...)
	}
}

func (l *Logger) Errorf(format string, v ...any) {
	if l.level <= LevelError {
		l.Printf("[ERROR] "+format, v...)
	}
}
