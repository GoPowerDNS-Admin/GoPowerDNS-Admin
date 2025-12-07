package logger_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/rs/zerolog/log"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/logger"
)

func TestLogger(t *testing.T) {
	type testCase struct {
		name             string
		cfg              logger.Log
		shouldHaveOutPut bool
		outPutIsJSON     bool
	}

	testCases := []testCase{
		{
			name: "no logger enabled log level not set",
			cfg: logger.Log{
				LogLevel:    "",
				ServiceName: "test",
				AppName:     "test",
			},
			shouldHaveOutPut: false,
		},
		{
			name: "console enabled log level info",
			cfg: logger.Log{
				LogLevel:    "info",
				ServiceName: "test",
				AppName:     "test",
				Console:     logger.Console{Enabled: true},
			},
			shouldHaveOutPut: true,
		},
		{
			name: "console enabled console writer enabled",
			cfg: logger.Log{
				LogLevel:    "info",
				ServiceName: "test",
				AppName:     "test",
				Console:     logger.Console{Enabled: true, UseConsoleWriter: true},
			},
			shouldHaveOutPut: true,
		},
		{
			name: "console enabled console writer enabled trace",
			cfg: logger.Log{
				LogLevel:    "trace",
				ServiceName: "test",
				AppName:     "test",
				Console:     logger.Console{Enabled: true, UseConsoleWriter: true},
			},
			shouldHaveOutPut: true,
		},
		{
			name: "console enabled console writer disabled info expect json",
			cfg: logger.Log{
				LogLevel:    "info",
				ServiceName: "test",
				AppName:     "test",
				Console:     logger.Console{Enabled: true, UseConsoleWriter: false},
			},
			shouldHaveOutPut: true,
			outPutIsJSON:     true,
		},
		{
			name: "console enabled console writer disabled trace expect json stack",
			cfg: logger.Log{
				LogLevel:     "trace",
				ServiceName:  "test",
				AppName:      "test",
				ReportCaller: true,
				Console:      logger.Console{Enabled: true, UseConsoleWriter: false},
			},
			shouldHaveOutPut: true,
			outPutIsJSON:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			out := testLoggerConfig(t, tc.cfg)
			t.Logf("out: %s", out)

			switch {
			case out == "" && tc.shouldHaveOutPut:
				t.Errorf("expected no console output but got: %s", out)
			case tc.outPutIsJSON:
				// split lines
				outSplit := strings.Split(out, "\n")
				// try to decode
				type Foo struct { //nolint:musttag
					Type    string
					Level   string
					Test    string
					Message string
				}

				dummy := Foo{}

				for _, outLine := range outSplit {
					if outLine != "" {
						if err := json.Unmarshal([]byte(outLine), &dummy); err != nil {
							t.Errorf("expected json output but got: %s", out) //nolint:goerr113
						} else {
							t.Log(dummy)
						}
					}
				}
			}
		})
	}
}

func alwaysErrFunc() error {
	return errors.New("a test error") //nolint:goerr113
}

func testLoggerConfig(t *testing.T, cfg logger.Log) string {
	t.Helper()
	// keep default std out
	stdout := os.Stdout
	stderr := os.Stderr

	// capture stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	os.Stderr = w

	err := logger.Init(cfg)
	if err != nil {
		t.Error(err)
	}

	log.Info().Msg("this info message should be seen...")
	log.Error().Err(alwaysErrFunc()).Msg("this err message should be seen...")
	log.Trace().Err(alwaysErrFunc()).Msg("this trace message should be seen...")

	outC := make(chan string)
	// copy the output in a separate goroutine so printing can't block indefinitely
	go func() {
		var buf bytes.Buffer
		_, err = io.Copy(&buf, r)
		if err != nil {
			t.Error(err)
		}
		outC <- buf.String()
	}()

	// back to normal state
	_ = w.Close()
	os.Stdout = stdout // restoring the real stdout
	os.Stderr = stderr // restoring the real stderr
	out := <-outC

	return out
}
