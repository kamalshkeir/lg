package lg

import (
	"fmt"
	"runtime"
	"strings"
	"time"
)

func (l *Logger) handle(level Level, ts time.Time, frames []runtime.Frame, msg any, keyvals ...any) {
	var kvs []any

	if level != NoLevel {
		kvs = append(kvs, LevelKey, level)
	}

	if l.reportCaller && len(frames) > 0 && frames[0].PC != 0 {
		file, line, fn := l.location(frames)
		if file != "" {
			caller := l.callerFormatter(file, line, fn)
			kvs = append(kvs, CallerKey, caller)
		}
	}

	if l.prefix != "" {
		kvs = append(kvs, PrefixKey, l.prefix)
	}

	if msg != nil {
		if m := fmt.Sprint(msg); m != "" {
			kvs = append(kvs, MessageKey, m)
		}
	}

	// append logger fields
	kvs = append(kvs, l.fields...)
	if len(l.fields)%2 != 0 {
		kvs = append(kvs, ErrMissingValue)
	}

	// append the rest
	kvs = append(kvs, keyvals...)
	if len(keyvals)%2 != 0 {
		kvs = append(kvs, ErrMissingValue)
	}

	if l.reportTimestamp && !ts.IsZero() {
		kvs = append(kvs, TimestampKey, ts)
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	switch l.formatter {
	case JSONFormatter:
		l.jsonFormatter(kvs...)
		// WriteTo will reset the buffer
		l.b.WriteTo(l.w) //nolint: errcheck
	default:
		l.textFormatter(kvs...)
	}
}

func (l *Logger) handleC(level Level, ts time.Time, frames []runtime.Frame, msg any, keyvals ...any) {
	var kvs []any

	if level != NoLevel {
		kvs = append(kvs, LevelKey, level)
	}

	if len(frames) > 0 && frames[0].PC != 0 {
		file, line, fn := l.location(frames)
		if file != "" {
			caller := l.callerFormatter(file, line, fn)
			kvs = append(kvs, CallerKey, caller)
		}
	}

	if l.prefix != "" {
		kvs = append(kvs, PrefixKey, l.prefix)
	}

	if msg != nil {
		if m := fmt.Sprint(msg); m != "" {
			kvs = append(kvs, MessageKey, m)
		}
	}

	// append logger fields
	kvs = append(kvs, l.fields...)
	if len(l.fields)%2 != 0 {
		kvs = append(kvs, ErrMissingValue)
	}

	// append the rest
	kvs = append(kvs, keyvals...)
	if len(keyvals)%2 != 0 {
		kvs = append(kvs, ErrMissingValue)
	}

	if l.reportTimestamp && !ts.IsZero() {
		kvs = append(kvs, TimestampKey, ts)
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	switch l.formatter {
	case JSONFormatter:
		l.jsonFormatter(kvs...)
		// WriteTo will reset the buffer
		l.b.WriteTo(l.w) //nolint: errcheck
	default:
		l.textFormatter(kvs...)
	}
}

func (l *Logger) helper(skip int) {
	var pcs [1]uintptr
	// Skip runtime.Callers, and l.helper
	n := runtime.Callers(skip+2, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])
	frame, _ := frames.Next()
	l.helpers.LoadOrStore(frame.Function, struct{}{})
}

// frames returns the runtime.Frames for the caller.
func (l *Logger) frames(skip int) *runtime.Frames {
	// Copied from testing.T
	const maxStackLen = 50
	var pc [maxStackLen]uintptr

	// Skip runtime.Callers, and l.frame
	n := runtime.Callers(skip+2, pc[:])
	frames := runtime.CallersFrames(pc[:n])
	return frames
}

func (l *Logger) location(frames []runtime.Frame) (file string, line int, fn string) {
	if len(frames) == 0 {
		return "", 0, ""
	}
	f := frames[0]
	return f.File, f.Line, f.Function
}

// Cleanup a path by returning the last n segments of the path only.
func trimCallerPath(path string, n int) string {
	// lovely borrowed from zap
	// nb. To make sure we trim the path correctly on Windows too, we
	// counter-intuitively need to use '/' and *not* os.PathSeparator here,
	// because the path given originates from Go stdlib, specifically
	// runtime.Caller() which (as of Mar/17) returns forward slashes even on
	// Windows.
	//
	// See https://github.com/golang/go/issues/3335
	// and https://github.com/golang/go/issues/18151
	//
	// for discussion on the issue on Go side.

	// Return the full path if n is 0.
	if n <= 0 {
		return path
	}

	// Find the last separator.
	idx := strings.LastIndexByte(path, '/')
	if idx == -1 {
		return path
	}

	for i := 0; i < n-1; i++ {
		// Find the penultimate separator.
		idx = strings.LastIndexByte(path[:idx], '/')
		if idx == -1 {
			return path
		}
	}

	return path[idx+1:]
}
