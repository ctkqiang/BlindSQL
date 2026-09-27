package utilities

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

type Level int

const (
	LevelTrace Level = iota
	LevelDebug
	LevelVerbose
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
	LevelPanic
)

var (
	levelNames  = [...]string{"TRACE", "DEBUG", "VERBOSE", "INFO", "WARN", "ERROR", "FATAL", "PANIC"}
	levelColors = [...]string{"\033[36m", "\033[34m", "\033[35m", "\033[32m", "\033[33m", "\033[31m", "\033[35;1m", "\033[41;37;1m"}
)

func (l Level) String() string {
	if int(l) >= 0 && int(l) < len(levelNames) {
		return levelNames[l]
	}
	return "UNKNOWN"
}

func ParseLevel(lvl string) (Level, error) {
	for i, name := range levelNames {
		if strings.EqualFold(name, strings.TrimSpace(lvl)) {
			return Level(i), nil
		}
	}
	return LevelInfo, fmt.Errorf("invalid log level: %s", lvl)
}

type Fields map[string]any

type Config struct {
	MinLevel     Level
	Output       io.Writer
	EnableColor  bool
	EnableCaller bool
	TimeFormat   string
}

type Logger struct {
	mu           sync.RWMutex
	minLevel     Level
	output       io.Writer
	enableColor  bool
	enableCaller bool
	timeFormat   string
	fields       Fields
}

func New(cfg Config) *Logger {
	if cfg.Output == nil {
		cfg.Output = os.Stdout
	}
	if cfg.TimeFormat == "" {
		cfg.TimeFormat = "2006-01-02 15:04:05.000"
	}
	return &Logger{
		minLevel:     cfg.MinLevel,
		output:       cfg.Output,
		enableColor:  cfg.EnableColor,
		enableCaller: cfg.EnableCaller,
		timeFormat:   cfg.TimeFormat,
		fields:       make(Fields),
	}
}

var Default = New(Config{MinLevel: LevelInfo, EnableColor: true, EnableCaller: true})

func (l *Logger) SetLevel(lvl Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.minLevel = lvl
}

func (l *Logger) WithFields(fields Fields) *Logger {
	l.mu.RLock()
	defer l.mu.RUnlock()
	newFields := make(Fields, len(l.fields)+len(fields))
	for k, v := range l.fields {
		newFields[k] = v
	}
	for k, v := range fields {
		newFields[k] = v
	}
	return &Logger{
		minLevel:     l.minLevel,
		output:       l.output,
		enableColor:  l.enableColor,
		enableCaller: l.enableCaller,
		timeFormat:   l.timeFormat,
		fields:       newFields,
	}
}

func (l *Logger) Log(lvl Level, msg string) {
	l.mu.RLock()
	if lvl < l.minLevel {
		l.mu.RUnlock()
		return
	}

	now := time.Now().Format(l.timeFormat)
	lvlStr := lvl.String()
	if l.enableColor && int(lvl) < len(levelColors) {
		lvlStr = fmt.Sprintf("%s%-7s\033[0m", levelColors[lvl], lvlStr)
	} else {
		lvlStr = fmt.Sprintf("%-7s", lvlStr)
	}

	var caller string
	if l.enableCaller {
		if _, file, line, ok := runtime.Caller(2); ok {
			caller = fmt.Sprintf(" %s:%d", filepath.Base(file), line)
		}
	}

	var sb strings.Builder
	if len(l.fields) > 0 {
		sb.WriteString(" |")
		for k, v := range l.fields {
			fmt.Fprintf(&sb, " %s=%v", k, v)
		}
	}
	l.mu.RUnlock()

	line := fmt.Sprintf("[%s] [%s]%s %s%s\n", now, lvlStr, caller, msg, sb.String())

	l.mu.Lock()
	fmt.Fprint(l.output, line)
	l.mu.Unlock()

	if lvl == LevelPanic {
		panic(msg)
	} else if lvl == LevelFatal {
		os.Exit(1)
	}
}

func (l *Logger) Logf(lvl Level, format string, v ...any) {
	l.Log(lvl, fmt.Sprintf(format, v...))
}

func (l *Logger) Trace(v ...any)                   { l.Log(LevelTrace, fmt.Sprint(v...)) }
func (l *Logger) Tracef(fmtStr string, v ...any)   { l.Logf(LevelTrace, fmtStr, v...) }
func (l *Logger) Debug(v ...any)                   { l.Log(LevelDebug, fmt.Sprint(v...)) }
func (l *Logger) Debugf(fmtStr string, v ...any)   { l.Logf(LevelDebug, fmtStr, v...) }
func (l *Logger) Verbose(v ...any)                 { l.Log(LevelVerbose, fmt.Sprint(v...)) }
func (l *Logger) Verbosef(fmtStr string, v ...any) { l.Logf(LevelVerbose, fmtStr, v...) }
func (l *Logger) Info(v ...any)                    { l.Log(LevelInfo, fmt.Sprint(v...)) }
func (l *Logger) Infof(fmtStr string, v ...any)    { l.Logf(LevelInfo, fmtStr, v...) }
func (l *Logger) Warn(v ...any)                    { l.Log(LevelWarn, fmt.Sprint(v...)) }
func (l *Logger) Warnf(fmtStr string, v ...any)    { l.Logf(LevelWarn, fmtStr, v...) }
func (l *Logger) Error(v ...any)                   { l.Log(LevelError, fmt.Sprint(v...)) }
func (l *Logger) Errorf(fmtStr string, v ...any)   { l.Logf(LevelError, fmtStr, v...) }
func (l *Logger) Fatal(v ...any)                   { l.Log(LevelFatal, fmt.Sprint(v...)) }
func (l *Logger) Fatalf(fmtStr string, v ...any)   { l.Logf(LevelFatal, fmtStr, v...) }
func (l *Logger) Panic(v ...any)                   { l.Log(LevelPanic, fmt.Sprint(v...)) }
func (l *Logger) Panicf(fmtStr string, v ...any)   { l.Logf(LevelPanic, fmtStr, v...) }
