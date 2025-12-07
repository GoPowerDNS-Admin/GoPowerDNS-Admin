package fiber_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"

	adapter "github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/logger/adapter/fiber"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/logger"
)

// expectedLoggerJSONFormat implements loggers default json format.
type expectedLoggerJSONFormat struct {
	IP            net.IP    `json:"IP"`
	Status        int       `json:"status"`
	XPerformance  float32   `json:"X-Performance"`
	URI           string    `json:"URI"`
	Method        string    `json:"method"`
	Host          string    `json:"host"`
	XForwardedFor string    `json:"X-Forwarded-For"`
	UserAgent     string    `json:"User-Agent"`
	Time          time.Time `json:"time"`
}

func TestNew(t *testing.T) {
	type arguments struct {
		config     adapter.Config
		targetPath string
	}

	type want struct {
		err    error
		output *expectedLoggerJSONFormat
	}

	tests := []struct {
		name string
		args arguments
		want want
	}{
		{
			name: "empty no output at all",
			args: arguments{
				targetPath: "/",
			},
			want: want{
				err:    nil,
				output: nil,
			},
		},
		{
			name: "get / log to console json",
			args: arguments{
				targetPath: "/",
				config: adapter.Config{
					Next: nil,
					Config: logger.Log{
						EnableAccessLogToConsole: true,
						Console:                  logger.Console{Enabled: true},
					},
					CacheControlError: "",
					CheckAliveURI:     "",
				},
			},
			want: want{
				err: nil,
				output: &expectedLoggerJSONFormat{
					IP:     net.ParseIP("0.0.0.0"),
					Status: 200,
					URI:    "/",
					Method: fiber.MethodGet,
					Host:   "example.com",
				},
			},
		},
		{
			name: "get multiples slash log to console json",
			args: arguments{
				targetPath: "//test",
				config: adapter.Config{
					Next: nil,
					Config: logger.Log{
						EnableAccessLogToConsole: true,
						Console:                  logger.Console{Enabled: true},
					},
					CacheControlError: "",
					CheckAliveURI:     "",
				},
			},
			want: want{
				err: nil,
				output: &expectedLoggerJSONFormat{
					IP:     net.ParseIP("0.0.0.0"),
					Status: 404,
					URI:    "//test",
					Method: fiber.MethodGet,
					Host:   "example.com",
				},
			},
		},
		{
			name: "get log with params",
			args: arguments{
				targetPath: "/?test=123",
				config: adapter.Config{
					Next: nil,
					Config: logger.Log{
						EnableAccessLogToConsole: true,
						Console:                  logger.Console{Enabled: true},
					},
					CacheControlError: "",
					CheckAliveURI:     "",
				},
			},
			want: want{
				err: nil,
				output: &expectedLoggerJSONFormat{
					IP:     net.ParseIP("0.0.0.0"),
					Status: 200,
					URI:    "/?test=123",
					Method: fiber.MethodGet,
					Host:   "example.com",
				},
			},
		},
		{
			name: "get multi slash and params",
			args: arguments{
				targetPath: "//?test=123",
				config: adapter.Config{
					Next: nil,
					Config: logger.Log{
						EnableAccessLogToConsole: true,
						Console:                  logger.Console{Enabled: true},
					},
					CacheControlError: "",
					CheckAliveURI:     "",
				},
			},
			want: want{
				err: nil,
				output: &expectedLoggerJSONFormat{
					IP:     net.ParseIP("0.0.0.0"),
					Status: 404,
					URI:    "//?test=123",
					Method: fiber.MethodGet,
					Host:   "example.com",
				},
			},
		},
		{
			name: "get multi slash 2 and params",
			args: arguments{
				targetPath: "/no_path//?test=123",
				config: adapter.Config{
					Next: nil,
					Config: logger.Log{
						EnableAccessLogToConsole: true,
						Console:                  logger.Console{Enabled: true},
					},
					CacheControlError: "",
					CheckAliveURI:     "",
				},
			},
			want: want{
				err: nil,
				output: &expectedLoggerJSONFormat{
					IP:     net.ParseIP("0.0.0.0"),
					Status: 404,
					URI:    "/no_path//?test=123",
					Method: fiber.MethodGet,
					Host:   "example.com",
				},
			},
		},
		{
			name: "get multi slash 2 and multi params",
			args: arguments{
				targetPath: "/2/customlist?metadataids=948925154001%2C535011865781%2C320846533135%2C857487483956%2C251159283476%2C690302913557%2C662584711062%2C811467381002%2C506151499580%2C153247415320%2C857963073722%2C357094445361%2C886450998030%2C450918246385%2C216412328956%2C895623245650%2C261928184649%2C208463789524%2C857754139329%2C226246233307%2C465770198994%2C419537784582%2C599911980596%2C342342848977%2C188036632324%2C578391149735%2C905806863695%2C450955988851%2C110674126988%2C732038917358%2C982375656878%2C585034470676%2C626547049540%2C628217944774%2C529855578166%2C102818516736%2C916552012229%2C771202171523%2C260687929078%2C663128946731%2C695475304451%2C484193452374%2C567241752605%2C577191897013%2C522354175551%2C880632090179%2C333671618832%2C559007432808%2C929820715552%2C171956595474%2C769954987746%2C897298225519%2C293952678746%2C153556486220%2C245149399406%2C504291501535%2C914447538583%2C600983173810%2C808797081276%2C770800526747%2C941025400063%2C582753818889%2C585061606649%2C630108930444%2C504381353289%2C517556078553%2C529082008404%2C192499167291%2C583210023858%2C616515990420%2C990836081941%2C120078704432%2C309774027444%2C225633248303%2C791439060221%2C739766849796%2C405346535881%2C399596308133%2C292641963159%2C921818853615%2C878337850529%2C223078318680%2C833271550285%2C466340039436%2C903234122334%2C481821539752%2C146920599813%2C632129662993%2C762590250695%2C549511640453%2C539358646362%2C798660398814%2C954445109028%2C857996554193%2C164695986793%2C287439858305%2C252789274895%2C293869206232%2C242301756689%2C111078020949", //nolint:lll
				config: adapter.Config{
					Next: nil,
					Config: logger.Log{
						EnableAccessLogToConsole: true,
						Console:                  logger.Console{Enabled: true},
					},
					CacheControlError: "",
					CheckAliveURI:     "",
				},
			},
			want: want{
				err: nil,
				output: &expectedLoggerJSONFormat{
					IP:     net.ParseIP("0.0.0.0"),
					Status: 404,
					URI:    "/2/customlist?metadataids=948925154001%2C535011865781%2C320846533135%2C857487483956%2C251159283476%2C690302913557%2C662584711062%2C811467381002%2C506151499580%2C153247415320%2C857963073722%2C357094445361%2C886450998030%2C450918246385%2C216412328956%2C895623245650%2C261928184649%2C208463789524%2C857754139329%2C226246233307%2C465770198994%2C419537784582%2C599911980596%2C342342848977%2C188036632324%2C578391149735%2C905806863695%2C450955988851%2C110674126988%2C732038917358%2C982375656878%2C585034470676%2C626547049540%2C628217944774%2C529855578166%2C102818516736%2C916552012229%2C771202171523%2C260687929078%2C663128946731%2C695475304451%2C484193452374%2C567241752605%2C577191897013%2C522354175551%2C880632090179%2C333671618832%2C559007432808%2C929820715552%2C171956595474%2C769954987746%2C897298225519%2C293952678746%2C153556486220%2C245149399406%2C504291501535%2C914447538583%2C600983173810%2C808797081276%2C770800526747%2C941025400063%2C582753818889%2C585061606649%2C630108930444%2C504381353289%2C517556078553%2C529082008404%2C192499167291%2C583210023858%2C616515990420%2C990836081941%2C120078704432%2C309774027444%2C225633248303%2C791439060221%2C739766849796%2C405346535881%2C399596308133%2C292641963159%2C921818853615%2C878337850529%2C223078318680%2C833271550285%2C466340039436%2C903234122334%2C481821539752%2C146920599813%2C632129662993%2C762590250695%2C549511640453%2C539358646362%2C798660398814%2C954445109028%2C857996554193%2C164695986793%2C287439858305%2C252789274895%2C293869206232%2C242301756689%2C111078020949", //nolint:lll
					Method: fiber.MethodGet,
					Host:   "example.com",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// use test helper func for testing this config
			output, err := testMiddlewareHelper(t, tt.args.targetPath, tt.args.config)

			// is error as expected
			assert.Equal(t, tt.want.err, err)

			if tt.want.output == nil && output != "" {
				t.Errorf("expected no output, but got output %s", output)
			}

			if tt.want.output != nil && output == "" {
				t.Error("expected output but got no output")
			}

			if tt.want.output != nil && output != "" {
				var decodedOutput expectedLoggerJSONFormat
				err = json.Unmarshal([]byte(output), &decodedOutput)
				if err != nil {
					t.Error(err)
					return
				}

				assert.Equal(t, tt.want.output.Host, decodedOutput.Host)
				assert.Equal(t, tt.want.output.Method, decodedOutput.Method)
				assert.Equal(t, tt.want.output.Status, decodedOutput.Status)
				assert.Equal(t, tt.want.output.IP, decodedOutput.IP)
				assert.Equal(t, tt.want.output.URI, decodedOutput.URI)
			}
		})
	}
}

func testMiddlewareHelper(t *testing.T, targetPath string, adapterConfig adapter.Config) (string, error) {
	t.Helper()

	stdout := os.Stdout
	stderr := os.Stderr

	// capture stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	os.Stderr = w

	// create new fiber app
	app := fiber.New(fiber.Config{
		CaseSensitive: true,
		Immutable:     true,
	})

	// use logger
	app.Use(adapter.New(adapterConfig))

	// create minimal endpoint
	app.Get("/", func(ctx *fiber.Ctx) error {
		return ctx.SendString("hello test")
	})

	_, err := app.Test(httptest.NewRequest(fiber.MethodGet, targetPath, nil), 100000)
	if err != nil {
		_ = w.Close()
		return "", err
	}

	outC := make(chan string)
	// copy the output in a separate goroutine so printing can't block indefinitely
	go func() {
		var buf bytes.Buffer
		_, err = io.Copy(&buf, r)
		if err != nil {
			return
		}

		outC <- buf.String()
	}()

	// back to normal state
	_ = w.Close()
	os.Stdout = stdout // restoring the real stdout
	os.Stderr = stderr // restoring the real stderr
	out := <-outC

	return out, err
}
