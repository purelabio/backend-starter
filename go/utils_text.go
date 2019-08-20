package main

import (
	"fmt"
	"strconv"
	"time"
)

func intToString(value int64) string {
	return strconv.FormatInt(value, 10)
}

func floatToString(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

/*
Returns a string of length N that consists of characters randomly chosen from
the given sample.

Inefficient, but supports Unicode characters.
*/
func randomCharSample(str string, count int) string {
	chars := []rune(str)
	buf := make([]rune, count)
	for i := range buf {
		buf[i] = chars[env.rand.Intn(len(chars))]
	}
	return string(buf)
}

func randomLetters(count int) string {
	return randomCharSample(LOWERCASE_LETTERS, count)
}

func formatTime(inst time.Time) string {
	return inst.Format(time.RFC3339)
}

func sprintDetailed(val interface{}) string {
	if val == nil {
		return ""
	}
	return fmt.Sprintf(`%+v`, val)
}

/*
TODO avoid converting the entire input to `[]rune`. Can be expensive for large
files.
*/
func sliceStringAsChars(str string, start int, end int) string {
	runes := []rune(str)

	if end >= 0 && end < len(runes) {
		runes = runes[:end]
	}

	if start >= 0 && start < len(runes) {
		runes = runes[start:]
	}

	return string(runes)
}

func strOr(vals ...string) string {
	for _, val := range vals {
		if val != "" {
			return val
		}
	}
	return ""
}
