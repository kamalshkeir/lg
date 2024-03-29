package lg

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// ErrMissingValue is returned when a key is missing a value.
var ErrMissingValue = fmt.Errorf("missing value")

// LoggerOption is an option for a logger.
type LoggerOption = func(*Logger)

// Logger is a Logger that implements Logger.
type Logger struct {
	w  io.Writer
	b  bytes.Buffer
	mu *sync.RWMutex
	re *lipgloss.Renderer

	isDiscard uint32

	level           int32
	prefix          string
	timeFunc        TimeFunction
	timeFormat      string
	callerOffset    int
	callerFormatter CallerFormatter
	formatter       Formatter

	reportCaller    bool
	reportTimestamp bool

	fields []any

	helpers *sync.Map
	styles  *Styles
}

// Logf logs a message with formatting.
func (l *Logger) Logf(level Level, format string, args ...any) {
	l.Log(level, fmt.Sprintf(format, args...))
}

// Log logs the given message with the given keyvals for the given level.
func (l *Logger) Log(level Level, msg any, keyvals ...any) {
	if atomic.LoadUint32(&l.isDiscard) != 0 {
		return
	}

	// check if the level is allowed
	if atomic.LoadInt32(&l.level) > int32(level) {
		return
	}

	var frame runtime.Frame
	if l.reportCaller {
		// Skip log.log, the caller, and any offset added.
		frames := l.frames(l.callerOffset + 2)
		for {
			f, more := frames.Next()
			_, helper := l.helpers.Load(f.Function)
			if !helper || !more {
				// Found a frame that wasn't a helper function.
				// Or we ran out of frames to check.
				frame = f
				break
			}
		}
	}
	l.handle(level, l.timeFunc(time.Now()), []runtime.Frame{frame}, msg, keyvals...)
}

func (l *Logger) LogC(level Level, msg any, keyvals ...any) {
	if atomic.LoadUint32(&l.isDiscard) != 0 {
		return
	}

	// check if the level is allowed
	if atomic.LoadInt32(&l.level) > int32(level) {
		return
	}

	var frame runtime.Frame
	frames := l.frames(l.callerOffset + 2)
	for {
		f, more := frames.Next()
		_, helper := l.helpers.Load(f.Function)
		if !helper || !more {
			// Found a frame that wasn't a helper function.
			// Or we ran out of frames to check.
			frame = f
			break
		}
	}
	l.handleC(level, l.timeFunc(time.Now()), []runtime.Frame{frame}, msg, keyvals...)
}

// Helper marks the calling function as a helper
// and skips it for source location information.
// It's the equivalent of testing.TB.Helper().
func (l *Logger) Helper() {
	l.helper(1)
}

// SetReportTimestamp sets whether the timestamp should be reported.
func (l *Logger) SetReportTimestamp(report bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.reportTimestamp = report
}

// SetReportCaller sets whether the caller location should be reported.
func (l *Logger) SetReportCaller(report bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.reportCaller = report
}

// GetLevel returns the current level.
func (l *Logger) GetLevel() Level {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return Level(l.level)
}

// SetLevel sets the current level.
func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	atomic.StoreInt32(&l.level, int32(level))
}

// GetPrefix returns the current prefix.
func (l *Logger) GetPrefix() string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.prefix
}

// SetPrefix sets the current prefix.
func (l *Logger) SetPrefix(prefix string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.prefix = prefix
}

// SetTimeFormat sets the time format.
func (l *Logger) SetTimeFormat(format string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.timeFormat = format
}

// SetTimeFunction sets the time function.
func (l *Logger) SetTimeFunction(f TimeFunction) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.timeFunc = f
}

// SetOutput sets the output destination.
func (l *Logger) SetOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if w == nil {
		w = os.Stderr
	}
	l.w = w
	var isDiscard uint32
	if w == io.Discard {
		isDiscard = 1
	}
	atomic.StoreUint32(&l.isDiscard, isDiscard)
	// Reuse cached renderers
	if v, ok := registry.Load(w); ok {
		l.re = v.(*lipgloss.Renderer)
	} else {
		l.re = lipgloss.NewRenderer(w, termenv.WithColorCache(true))
		registry.Store(w, l.re)
	}
}

// SetFormatter sets the formatter.
func (l *Logger) SetFormatter(f Formatter) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.formatter = f
}

// SetCallerFormatter sets the caller formatter.
func (l *Logger) SetCallerFormatter(f CallerFormatter) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.callerFormatter = f
}

// SetCallerOffset sets the caller offset.
func (l *Logger) SetCallerOffset(offset int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.callerOffset = offset
}

// SetColorProfile force sets the underlying Lip Gloss renderer color profile
// for the TextFormatter.
func (l *Logger) SetColorProfile(profile termenv.Profile) {
	l.re.SetColorProfile(profile)
}

// SetStyles sets the logger styles for the TextFormatter.
func (l *Logger) SetStyles(s *Styles) {
	if s == nil {
		s = DefaultStyles()
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.styles = s
}

// With returns a new logger with the given keyvals added.
func (l *Logger) With(keyvals ...any) *Logger {
	var st Styles
	l.mu.Lock()
	sl := *l
	st = *l.styles
	l.mu.Unlock()
	sl.b = bytes.Buffer{}
	sl.mu = &sync.RWMutex{}
	sl.helpers = &sync.Map{}
	sl.fields = append(l.fields, keyvals...)
	sl.styles = &st
	return &sl
}

// WithPrefix returns a new logger with the given prefix.
func (l *Logger) WithPrefix(prefix string) *Logger {
	sl := l.With()
	sl.SetPrefix(prefix)
	return sl
}

// Debug prints a debug message.
func (l *Logger) Debug(msg any, keyvals ...any) {
	l.Log(DebugLevel, msg, keyvals...)
}

// Info prints an info message.
func (l *Logger) Info(msg any, keyvals ...any) {
	l.Log(InfoLevel, msg, keyvals...)
}

// Warn prints a warning message.
func (l *Logger) Warn(msg any, keyvals ...any) {
	l.Log(WarnLevel, msg, keyvals...)
}

// Error prints an error message.
func (l *Logger) Error(msg any, keyvals ...any) {
	l.Log(ErrorLevel, msg, keyvals...)
}

// Fatal prints a fatal message and exits.
func (l *Logger) Fatal(msg any, keyvals ...any) {
	l.Log(FatalLevel, msg, keyvals...)
	os.Exit(1)
}

// Print prints a message with no level.
func (l *Logger) Print(msg any, keyvals ...any) {
	l.Log(NoLevel, msg, keyvals...)
}

// Debugf prints a debug message with formatting.
func (l *Logger) Debugf(format string, args ...any) {
	l.Log(DebugLevel, fmt.Sprintf(format, args...))
}

// Infof prints an info message with formatting.
func (l *Logger) Infof(format string, args ...any) {
	l.Log(InfoLevel, fmt.Sprintf(format, args...))
}

// Warnf prints a warning message with formatting.
func (l *Logger) Warnf(format string, args ...any) {
	l.Log(WarnLevel, fmt.Sprintf(format, args...))
}

// Errorf prints an error message with formatting.
func (l *Logger) Errorf(format string, args ...any) {
	l.Log(ErrorLevel, fmt.Sprintf(format, args...))
}

// Fatalf prints a fatal message with formatting and exits.
func (l *Logger) Fatalf(format string, args ...any) {
	l.Log(FatalLevel, fmt.Sprintf(format, args...))
	os.Exit(1)
}

// Printf prints a message with no level and formatting.
func (l *Logger) Printf(format string, args ...any) {
	l.Log(NoLevel, fmt.Sprintf(format, args...))
}
