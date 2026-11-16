package colorlog

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// Level defines severity.
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	default:
		return "INFO"
	}
}

// ANSI Escape Codes
const (
	resetCode   = "\x1b[0m"
	boldCode    = "\x1b[1m"
	grayCode    = "\x1b[90m"
	cyanCode    = "\x1b[36m"
	greenCode   = "\x1b[32m"
	yellowCode  = "\x1b[33m"
	redCode     = "\x1b[31m"
	magentaCode = "\x1b[35m"
	boldRedCode = "\x1b[1;31m"
)

var bufPool = sync.Pool{
	New: func() any {
		return bytes.NewBuffer(make([]byte, 0, 512))
	},
}

// Logger formats and outputs structured logs.
type Logger struct {
	mu            sync.Mutex
	out           io.Writer
	level         Level
	contextFields []any
	showCaller    bool
	noColor       bool
}

// Option configures a Logger.
type Option func(*Logger)

// WithOutput sets the destination writer.
func WithOutput(w io.Writer) Option {
	return func(l *Logger) {
		if w != nil {
			l.out = w
		}
	}
}

// WithLevel sets the minimum enabled log level.
func WithLevel(lvl Level) Option {
	return func(l *Logger) {
		l.level = lvl
	}
}

// WithCaller enables/disables caller file and line number.
func WithCaller(enabled bool) Option {
	return func(l *Logger) {
		l.showCaller = enabled
	}
}

// WithNoColor explicitly forces color disabling.
func WithNoColor(noColor bool) Option {
	return func(l *Logger) {
		l.noColor = noColor
	}
}

// New creates a configured Logger.
func New(opts ...Option) *Logger {
	l := &Logger{
		out:        os.Stdout,
		level:      LevelInfo,
		showCaller: true,
		noColor:    os.Getenv("NO_COLOR") != "",
	}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

// Default is the package-level default logger.
var Default = New()

// SetLevel updates the minimum active level.
func (l *Logger) SetLevel(lvl Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = lvl
}

// With returns a child logger carrying persistent context key-value pairs.
func (l *Logger) With(keyvals ...any) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	child := &Logger{
		out:           l.out,
		level:         l.level,
		showCaller:    l.showCaller,
		noColor:       l.noColor,
		contextFields: make([]any, len(l.contextFields)+len(keyvals)),
	}
	copy(child.contextFields, l.contextFields)
	copy(child.contextFields[len(l.contextFields):], keyvals)
	return child
}

func (l *Logger) log(lvl Level, msg string, keyvals []any) {
	if lvl < l.level {
		return
	}

	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufPool.Put(buf)

	// 1. Timestamp
	now := time.Now().Format("2006-01-02 15:04:05.000")
	if !l.noColor {
		buf.WriteString(grayCode)
		buf.WriteString(now)
		buf.WriteString(resetCode)
	} else {
		buf.WriteString(now)
	}
	buf.WriteByte(' ')

	// 2. Level Badge
	if !l.noColor {
		switch lvl {
		case LevelDebug:
			buf.WriteString(magentaCode)
		case LevelInfo:
			buf.WriteString(greenCode)
		case LevelWarn:
			buf.WriteString(yellowCode)
		case LevelError:
			buf.WriteString(redCode)
		case LevelFatal:
			buf.WriteString(boldRedCode)
		}
		buf.WriteString(fmt.Sprintf("[%s]", lvl.String()))
		buf.WriteString(resetCode)
	} else {
		buf.WriteString(fmt.Sprintf("[%s]", lvl.String()))
	}
	buf.WriteByte(' ')

	// 3. Caller Info
	if l.showCaller {
		_, file, line, ok := runtime.Caller(3)
		if ok {
			fileShort := filepath.Base(file)
			if !l.noColor {
				buf.WriteString(grayCode)
				buf.WriteString(fmt.Sprintf("[%s:%d]", fileShort, line))
				buf.WriteString(resetCode)
			} else {
				buf.WriteString(fmt.Sprintf("[%s:%d]", fileShort, line))
			}
			buf.WriteByte(' ')
		}
	}

	// 4. Message
	if !l.noColor {
		buf.WriteString(boldCode)
		buf.WriteString(msg)
		buf.WriteString(resetCode)
	} else {
		buf.WriteString(msg)
	}

	// 5. Fields (context fields + call fields)
	allFields := l.contextFields
	if len(keyvals) > 0 {
		allFields = append(append([]any(nil), l.contextFields...), keyvals...)
	}

	for i := 0; i < len(allFields); i += 2 {
		buf.WriteByte(' ')
		key := fmt.Sprint(allFields[i])
		var val any = "<missing>"
		if i+1 < len(allFields) {
			val = allFields[i+1]
		}

		if !l.noColor {
			buf.WriteString(cyanCode)
			buf.WriteString(key)
			buf.WriteString(resetCode)
			buf.WriteByte('=')
			buf.WriteString(fmt.Sprint(val))
		} else {
			buf.WriteString(fmt.Sprintf("%s=%v", key, val))
		}
	}

	buf.WriteByte('\n')

	l.mu.Lock()
	defer l.mu.Unlock()
	_, _ = l.out.Write(buf.Bytes())
}

// Debug logs at DEBUG level.
func (l *Logger) Debug(msg string, keyvals ...any) { l.log(LevelDebug, msg, keyvals) }

// Info logs at INFO level.
func (l *Logger) Info(msg string, keyvals ...any) { l.log(LevelInfo, msg, keyvals) }

// Warn logs at WARN level.
func (l *Logger) Warn(msg string, keyvals ...any) { l.log(LevelWarn, msg, keyvals) }

// Error logs at ERROR level.
func (l *Logger) Error(msg string, keyvals ...any) { l.log(LevelError, msg, keyvals) }

// Package-level forwarders:

func Debug(msg string, keyvals ...any) { Default.Debug(msg, keyvals...) }
func Info(msg string, keyvals ...any)  { Default.Info(msg, keyvals...) }
func Warn(msg string, keyvals ...any)  { Default.Warn(msg, keyvals...) }
func Error(msg string, keyvals ...any) { Default.Error(msg, keyvals...) }
func With(keyvals ...any) *Logger      { return Default.With(keyvals...) }
