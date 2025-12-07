package stdlogger_test

import (
	"bytes"
	"errors"
	"io"
	"os"
	"runtime"
	"testing"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/logger/adapter/stdlogger"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/logger"
)

func TestNew(t *testing.T) {
	err := logger.Init(logger.Log{
		LogLevel:          "info",
		LogEnv:            "test",
		ReportCaller:      false,
		DisableCheckAlive: false,
		AppName:           "test",
		ServiceName:       "test",
		Console: logger.Console{
			Enabled:          true,
			UseConsoleWriter: true,
		},
		File:   logger.LogFile{},
		LogZio: logger.LogZio{},
	})
	if err != nil {
		t.Error(err)
	}

	testLogger := stdlogger.New()

	t.Log("zerolog stdlogger with InfoLevel was successfully created. No Debug should be shown")
	arch := runtime.GOARCH
	// this 3 log messages should be shown...
	testLogger.Infof("%s: this testLogger implements Infof()", arch)
	testLogger.Errorf("%v: this testLogger implements Errorf()", errors.New("this a generic error")) //nolint:goerr113
	testLogger.Warningf("%d: this testLogger implements Warningf()", runtime.NumCPU())

	// this one not
	testLogger.Debugf("%s: this testLogger implements Debugf()", runtime.GOOS)
}

func TestAdapter(t *testing.T) {
	type testCase struct {
		name             string
		cfg              logger.Log
		shouldHaveOutPut bool
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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			out := testLoggerConfig(t, tc.cfg)
			t.Logf("out: %s", out)
			if out == "" && tc.shouldHaveOutPut {
				t.Errorf("expected no console output but got: %s", out)
			}
		})
	}
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

	testLogger := stdlogger.New()

	testLogger.Debugf("stdloggger %s", "test debug")
	testLogger.Infof("stdloggger %s", "test info")
	testLogger.Warningf("stdloggger %s", "test warning")
	testLogger.Errorf("stdloggger %s", "test error")

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
