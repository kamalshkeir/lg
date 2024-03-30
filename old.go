package lg

import (
	"fmt"
)

const (
	GRAY    = "\033[1;30m%v\033[0m"
	RED     = "\033[1;31m%v\033[0m"
	GREEN   = "\033[1;32m%v\033[0m"
	YELLOW  = "\033[1;33m%v\033[0m"
	BLUE    = "\033[1;34m%v\033[0m"
	MAGENTA = "\033[1;35m%v\033[0m"
	AQUA    = "\033[1;36m%v\033[0m"
)

func Printfs(pattern string, anything ...interface{}) {
	var colorCode string
	var colorUsed = true
	switch pattern[:2] {
	case "gy":
		colorCode = "30"
	case "rd":
		colorCode = "31"
	case "gr":
		colorCode = "32"
	case "yl":
		colorCode = "33"
	case "bl":
		colorCode = "34"
	case "mg":
		colorCode = "35"
	default:
		colorUsed = false
	}
	if colorUsed {
		pattern = pattern[2:]
	}
	msg := fmt.Sprintf(pattern, anything...)
	ss.Add(msg)
	if pub != nil {
		pub.Publish(topicPub, map[string]any{
			"log": msg,
		})
	}
	if colorUsed {
		msg = "\033[1;" + colorCode + "m" + msg + "\033[0m"
	}
	fmt.Fprint(Default().w, msg)
}
