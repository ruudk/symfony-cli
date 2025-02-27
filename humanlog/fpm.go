package humanlog

import (
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// [12-Aug-2020 16:34:44] NOTICE: Terminating ...
var fpmLogLineRegexp = regexp.MustCompile(`^\[(.+?)\] (DEBUG|NOTICE|WARNING|ERROR|ALERT)\:((?: *?PHP (?:.+?)\:)*) (.+)\s*$`)

func convertPHPFPMLog(in []byte) (*line, error) {
	allMatches := fpmLogLineRegexp.FindAllStringSubmatch(string(in), -1)
	if allMatches == nil {
		return nil, nil
	}

	level := strings.ToLower(allMatches[0][2])

	// FPM will log as notice warning or fatal errors at PHP level, we need to
	// parse them and restore this level
	if subs := allMatches[0][3]; subs != "" {
		for _, sub := range strings.Split(subs, ":") {
			sub = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(sub)), "php ")
			switch sub {
			case "notice", "warning", "error", "fatal", "panic", "critical", "emergency":
				level = sub
			case "warn":
				level = "warning"
			case "fatal error":
				level = "fatal"
			}
		}
	}

	line := &line{
		source:  "FPM",
		level:   level,
		message: allMatches[0][4],
		fields:  make(map[string]string),
	}
	// convert date (Wed Aug 12 16:39:56 2020)
	var err error
	line.time, err = time.Parse(`2-Jan-2006 15:04:05`, allMatches[0][1])
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return line, nil
}
