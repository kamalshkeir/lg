package lg

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	separator       = "="
	indentSeparator = "  │ "
)

func (l *Logger) writeIndent(w io.Writer, str string, indent string, newline bool) {
	for {
		nl := strings.IndexByte(str, '\n')
		if nl == -1 {
			if str != "" {
				_, _ = w.Write([]byte(indent))
				val := escapeStringForOutput(str, false)
				_, _ = w.Write([]byte(val))
				if newline {
					_, _ = w.Write([]byte{'\n'})
				}
			}
			return
		}

		_, _ = w.Write([]byte(indent))
		val := escapeStringForOutput(str[:nl], false)
		_, _ = w.Write([]byte(val))
		_, _ = w.Write([]byte{'\n'})
		str = str[nl+1:]
	}
}

func needsEscaping(str string) bool {
	for _, b := range str {
		if !unicode.IsPrint(b) || b == '"' {
			return true
		}
	}

	return false
}

const (
	lowerhex = "0123456789abcdef"
)

var bufPool = sync.Pool{
	New: func() any {
		return new(strings.Builder)
	},
}

func escapeStringForOutput(str string, escapeQuotes bool) string {
	// kindly borrowed from hclog
	if !needsEscaping(str) {
		return str
	}

	bb := bufPool.Get().(*strings.Builder)
	bb.Reset()

	defer bufPool.Put(bb)
	for _, r := range str {
		if escapeQuotes && r == '"' {
			bb.WriteString(`\"`)
		} else if unicode.IsPrint(r) {
			bb.WriteRune(r)
		} else {
			switch r {
			case '\a':
				bb.WriteString(`\a`)
			case '\b':
				bb.WriteString(`\b`)
			case '\f':
				bb.WriteString(`\f`)
			case '\n':
				bb.WriteString(`\n`)
			case '\r':
				bb.WriteString(`\r`)
			case '\t':
				bb.WriteString(`\t`)
			case '\v':
				bb.WriteString(`\v`)
			default:
				switch {
				case r < ' ':
					bb.WriteString(`\x`)
					bb.WriteByte(lowerhex[byte(r)>>4])
					bb.WriteByte(lowerhex[byte(r)&0xF])
				case !utf8.ValidRune(r):
					r = 0xFFFD
					fallthrough
				case r < 0x10000:
					bb.WriteString(`\u`)
					for s := 12; s >= 0; s -= 4 {
						bb.WriteByte(lowerhex[r>>uint(s)&0xF])
					}
				default:
					bb.WriteString(`\U`)
					for s := 28; s >= 0; s -= 4 {
						bb.WriteByte(lowerhex[r>>uint(s)&0xF])
					}
				}
			}
		}
	}

	return bb.String()
}

func needsQuoting(s string) bool {
	for i := 0; i < len(s); {
		b := s[i]
		if b < utf8.RuneSelf {
			if needsQuotingSet[b] {
				return true
			}
			i++
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError || unicode.IsSpace(r) || !unicode.IsPrint(r) {
			return true
		}
		i += size
	}
	return false
}

var needsQuotingSet = [utf8.RuneSelf]bool{
	'"': true,
	'=': true,
}

func init() {
	for i := 0; i < utf8.RuneSelf; i++ {
		r := rune(i)
		if unicode.IsSpace(r) || !unicode.IsPrint(r) {
			needsQuotingSet[i] = true
		}
	}
}

func writeSpace(w io.Writer, first bool) {
	if !first {
		w.Write([]byte{' '}) //nolint: errcheck
	}
}

func Render(str, color string) string {
	switch color {
	case "gy", "gray":
		return fmt.Sprintf(GRAY, str)
	case "gr", "green":
		return fmt.Sprintf(GREEN, str)
	case "bl", "blue":
		return fmt.Sprintf(BLUE, str)
	case "rd", "red":
		return fmt.Sprintf(RED, str)
	case "mg", "magenta":
		return fmt.Sprintf(MAGENTA, str)
	case "aq", "aqua":
		return fmt.Sprintf(MAGENTA, str)
	default:
		return str
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
	for i := 0; i < lenKeyvals; i += 2 {
		firstKey := i == 0
		moreKeys := i < lenKeyvals-2
		switch keyvals[i] {
		case TimestampKey:
			if t, ok := keyvals[i+1].(time.Time); ok {
				ts := t.Format(l.timeFormat)
				pubMessage += "  " + TimestampKey + "=" + ts
				ts = Render(TimestampKey+"=", "gy") + ts
				writeSpace(&l.b, firstKey)
				l.b.WriteString(ts)
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
					lvl = Render(lvl, cc)
					writeSpace(&l.b, firstKey)
					l.b.WriteString(lvl)
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
				caller = Render(caller, "gy")
				writeSpace(&l.b, firstKey)
				l.b.WriteString(caller)
			}
		case PrefixKey:
			if prefix, ok := keyvals[i+1].(string); ok {
				pubMessage += " " + prefix + ":"
				prefix = Render(prefix+":", "gy")
				writeSpace(&l.b, firstKey)
				l.b.WriteString(prefix)
			}
		case MessageKey:
			if msg := keyvals[i+1]; msg != nil {
				m := fmt.Sprint(msg)
				pubMessage += " " + m
				writeSpace(&l.b, firstKey)
				l.b.WriteString(m)
			}
		default:
			sep := separator
			indentSep := indentSeparator
			sep = Render(sep, "gy")
			// indentSep = st.Separator.Renderer(l.re).Render(indentSep)
			key := fmt.Sprint(keyvals[i])
			val := fmt.Sprintf("%+v", keyvals[i+1])
			pubMessage += " " + key + "=" + val
			key = Render(key, "gy")
			raw := val == ""
			if raw {
				val = `""`
			}
			if key == "" {
				continue
			}

			// Values may contain multiple lines, and that format
			// is preserved, with each line prefixed with a "  | "
			// to show it's part of a collection of lines.
			//
			// Values may also need quoting, if not all the runes
			// in the value string are "normal", like if they
			// contain ANSI escape sequences.
			if strings.Contains(val, "\n") {
				l.b.WriteString("\n  ")
				l.b.WriteString(key)
				l.b.WriteString(sep + "\n")
				l.writeIndent(&l.b, val, indentSep, moreKeys)
			} else if !raw && needsQuoting(val) {
				writeSpace(&l.b, firstKey)
				l.b.WriteString(key)
				l.b.WriteString(sep)
				l.b.WriteString(fmt.Sprintf(`"%s"`,
					escapeStringForOutput(val, true)))
			} else {
				writeSpace(&l.b, firstKey)
				l.b.WriteString(key)
				l.b.WriteString(sep)
				l.b.WriteString(val)
			}
		}
	}
	if pub != nil {
		ss.Add(pubMessage)
		pub.Publish(topicPub, map[string]any{
			"log": pubMessage,
		})
	}
	// Add a newline to the end of the log message.
	l.b.WriteByte('\n')
}
