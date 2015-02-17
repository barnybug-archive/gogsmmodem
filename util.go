package gogsmmodem

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"io"
)

// Time format in AT protocol
var TimeFormat = "06/01/02,15:04:05"

// Parse an AT formatted time
func parseTime(t string) time.Time {
	t = t[:len(t)-3] // ignore trailing +00
	ret, _ := time.Parse(TimeFormat, t)
	return ret
}

// Quote a value
func quote(s interface{}) string {
	switch v := s.(type) {
	case string:
		if v == "?" {
			return v
		}
		return fmt.Sprintf(`"%s"`, v)
	case int, int64:
		return fmt.Sprint(v)
	default:
		panic(fmt.Sprintf("Unsupported argument type: %T", v))
	}
	return ""
}

// Quote a list of values
func quotes(args []interface{}) string {
	ret := make([]string, len(args))
	for i, arg := range args {
		ret[i] = quote(arg)
	}
	return strings.Join(ret, ",")
}

// Check if s starts with p
func startsWith(s, p string) bool {
	return strings.Index(s, p) == 0
}

// Unquote a string to a value (string or int)
func unquote(s string) interface{} {
	if startsWith(s, `"`) {
		return strings.Trim(s, `"`)
	}
	if i, err := strconv.Atoi(s); err == nil {
		// number
		return i
	}
	return s
}

var RegexQuote = regexp.MustCompile(`"[^"]*"|[^,]*`)

// Unquote a parameter list to values
func unquotes(s string) []interface{} {
	vs := RegexQuote.FindAllString(s, -1)
	args := make([]interface{}, len(vs))
	for i, v := range vs {
		args[i] = unquote(v)
	}
	return args
}

// Unquote a parameter list of strings
func stringsUnquotes(s string) []string {
	args := unquotes(s)
	var res []string
	for _, arg := range args {
		res = append(res, fmt.Sprint(arg))
	}
	return res
}

// A logging ReadWriteCloser for debugging
type LogReadWriteCloser struct {
	f io.ReadWriteCloser
}

func (self LogReadWriteCloser) Read(b []byte) (int, error) {
	n, err := self.f.Read(b)
	log.Printf("Read(%#v) = (%d, %v)\n", string(b[:n]), n, err)
	return n, err
}

func (self LogReadWriteCloser) Write(b []byte) (int, error) {
	n, err := self.f.Write(b)
	log.Printf("Write(%#v) = (%d, %v)\n", string(b), n, err)
	return n, err
}

func (self LogReadWriteCloser) Close() error {
	err := self.f.Close()
	log.Printf("Close() = %v\n", err)
	return err
}
