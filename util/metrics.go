package util

import (
	"fmt"
	"log"
	"strings"
	"time"
)

// Count parses a string into a count for logging to librato
func Count(metric string) { CountMany(metric, 1) }

// CountMany parses a string and number into a count for logging to librato
func CountMany(metric string, count int64) { CountWithData(metric, count, "") }

// CountWithData parses metrics for logging to librato
func CountWithData(metric string, count int64, extraData string, v ...interface{}) {
	if extraData == "" {
		log.Printf("count#busl.%s=%d", metric, count)
	} else {
		log.Printf("count#busl.%s=%d %s", metric, count, fmt.Sprintf(extraData, v...))
	}
}

func Sample(metric string, value int64) { SampleWithData(metric, value, "") }

func SampleWithData(metric string, value int64, extraData string, v ...interface{}) {
	if extraData == "" {
		log.Printf("sample#busl.%s=%d", metric, value)
	} else {
		log.Printf("sample#busl.%s=%d %s", metric, value, fmt.Sprintf(extraData, v...))
	}
}

func SMeasure(subject string, object string) string {
	return fmt.Sprintf("measure#%s.%s=%s", prefix, subject, object)
}

func TimerStart(subject string, extras ...string) (time.Time, string, []string) {
	log.Printf("%s.timer.start %s", subject, strings.Join(extras, " "))
	return time.Now(), subject, extras
}

func TimerEnd(startTime time.Time, subject string, extras []string) {
	elapsed := fmt.Sprintf("%f", time.Now().Sub(startTime).Seconds())
	log.Printf("%s.timer.end %s %s", subject, SMeasure(subject+".elapsed.seconds", elapsed), strings.Join(extras, " "))
}
