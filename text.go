package lg

import (
	"fmt"
	"io"
	"time"
)

const (
	separator = "="
)

func writeSpace(w io.Writer, first bool) {
	if !first {
		w.Write([]byte{' '}) //nolint: errcheck
	}
}

func Color(color string) string {
	switch color {
	case "gy", "gray":
		return GRAY
	case "gr", "green":
		return GREEN
	case "yl", "yellow":
		return YELLOW
	case "bl", "blue":
		return BLUE
	case "rd", "red":
		return RED
	case "mg", "magenta":
		return MAGENTA
	case "aq", "aqua":
		return AQUA
	default:
		ErrorC("color not found")
		return ""
	}
}

func toCapLevel(level string) string {
	switch level {
	case "debug":
		return "DEBUG"
	case "info":
		return "INFO"
	case "warn":
		return "WARN"
	case "error":
		return "ERROR"
	case "fatal":
		return "FATAL"
	default:
		return level
	}
}

func (l *Logger) textFormatter(keyvals ...any) {
	lenKeyvals := len(keyvals)
	pubMessage := ""
	args := make([]any, 0, len(keyvals)/2)
	defer l.b.Reset()
	for i := 0; i < lenKeyvals; i += 2 {
		firstKey := i == 0
		switch keyvals[i] {
		case TimestampKey:
			if t, ok := keyvals[i+1].(time.Time); ok {
				ts := t.Format(l.timeFormat)
				pubMessage += "  " + TimestampKey + "=" + ts
				color := Color("gy") + ts
				writeSpace(&l.b, firstKey)
				l.b.WriteString(color)
				args = append(args, TimestampKey+"=")
			}
		case LevelKey:
			if level, ok := keyvals[i+1].(Level); ok {
				lvl := level.String()
				if len(lvl) > 0 {
					cc := levelColors[lvl]
					lvl = toCapLevel(lvl)
					if len(lvl) > 3 {
						lvl = lvl[:4]
					}
					if pubMessage == "" {
						pubMessage += lvl
					} else {
						pubMessage += " " + lvl
					}
					color := Color(cc)
					writeSpace(&l.b, firstKey)
					l.b.WriteString(color)
					args = append(args, lvl)
				}
			}
		case CallerKey:
			if caller, ok := keyvals[i+1].(string); ok {
				caller = "[" + caller + "]"
				if pubMessage == "" {
					pubMessage += caller
				} else {
					pubMessage += " " + caller
				}
				cc := Color("gy")
				writeSpace(&l.b, firstKey)
				l.b.WriteString(cc)
				args = append(args, caller)
			}
		case PrefixKey:
			if prefix, ok := keyvals[i+1].(string); ok {
				pubMessage += " " + prefix + ":"
				cc := Color("gy")
				writeSpace(&l.b, firstKey)
				l.b.WriteString(cc)
				args = append(args, prefix+":")
			}
		case MessageKey:
			if msg := keyvals[i+1]; msg != nil {
				if v, ok := msg.(string); ok {
					pubMessage += " " + v
					writeSpace(&l.b, firstKey)
					l.b.WriteString(v)
				} else {
					fmt.Println("expect msg to be string, got:", msg)
					return
				}
			}
		default:
			sep := separator
			key := keyvals[i]
			if vStr, ok := key.(string); !ok {
				fmt.Printf("log error: textFormatter, expect key to be string, got %v\n", key)
				return
			} else {
				key = vStr + sep
				cc := " " + Color("gy")
				val := fmt.Sprintf("%+v", keyvals[i+1])
				raw := val == ""
				if raw {
					val = `""`
				}
				if vStr == "" {
					continue
				}
				pubMessage += " " + vStr + "=" + val
				l.b.WriteString(cc + val)
				args = append(args, key)
			}
		}
	}
	if saveMem {
		ss.Add(pubMessage)
	}
	if usePub && pub != nil {
		pub.Publish(topicPub, map[string]any{
			"log": pubMessage,
		})
	}
	_, err := fmt.Fprintf(l.w, l.b.String()+"\n", args...)
	l.CheckError(err)
}
