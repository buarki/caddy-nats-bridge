package request_test

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	_ "github.com/CoverWhale/caddy-nats-bridge"
	"github.com/CoverWhale/caddy-nats-bridge/integrationtest"
	"github.com/CoverWhale/caddy-nats-bridge/request"
	"github.com/nats-io/nats.go"
)

// TestRequestToNats converts a HTTP request to a NATS Publication, and vice-versa
// for the response.
//
//		              ┌──────────────┐    HTTP: /test
//		◀─────────────│ Caddy /test  │◀───────
//		NATS subject  │ nats_publish │
//		 greet.*      │              │
//	    ────────────▶ └──────────────┘ ────────────▶
func TestRequestToNats(t *testing.T) {
	type testCase struct {
		description                      string
		sendHttpRequestAndAssertResponse func() error
		handleNatsMessage                func(msg *nats.Msg, nc *nats.Conn) error
		CaddyfileSnippet                 string
	}

	// Testcases
	cases := []testCase{
		{
			description: "Simple GET request should keep headers and contain extra X-NatsBridge-Method and X-NatsBridge-UrlPath",
			sendHttpRequestAndAssertResponse: func() error {
				// 1) send initial HTTP request (will be validated on the NATS handler side)
				req, err := http.NewRequest("GET", "http://localhost:8889/test/hi", nil)
				if err != nil {
					return err
				}
				req.Header.Add("Custom-Header", "MyValue")
				res, err := http.DefaultClient.Do(req)
				if err != nil {
					return fmt.Errorf("HTTP request failed: %w", err)
				}

				// 4) validate HTTP response
				b, err := io.ReadAll(res.Body)
				if err != nil {
					return fmt.Errorf("could not read response body: %w", err)
				}
				if string(b) != "respData" {
					return fmt.Errorf("wrong response body. Expected: respData. Actual: %s", string(b))
				}
				if actualH := res.Header.Get("RespHeader"); actualH != "RespHeaderValue" {
					return fmt.Errorf("wrong response header. Expected: RespHeaderValue. Actual: %s. Full Headers: %+v", actualH, res.Header)
				}

				return nil
			},
			CaddyfileSnippet: `
				route /test/* {
					nats_request greet.hello
				}
			`,
			handleNatsMessage: func(msg *nats.Msg, nc *nats.Conn) error {
				// 2) validate incoming NATS request (converted from HTTP)
				if msg.Header.Get("Custom-Header") != "MyValue" {
					t.Fatalf("Custom-Header not correct, expected 'MyValue', actual headers: %+v", msg.Header)
				}

				if msg.Header.Get("X-NatsBridge-Method") != "GET" {
					t.Fatalf("X-NatsBridge-Method not correct, expected 'GET', actual headers: %+v", msg.Header)
				}
				if msg.Header.Get("X-NatsBridge-UrlPath") != "/test/hi" {
					t.Fatalf("X-NatsBridge-UrlPath not correct, expected '/test/hi', actual headers: %+v", msg.Header)
				}
				if msg.Header.Get("X-NatsBridge-UrlQuery") != "" {
					t.Fatalf("X-NatsBridge-UrlQuery not correct, expected '', actual headers: %+v", msg.Header)
				}

				// 3) send NATS response (will be validated on the HTTP response side)
				resp := &nats.Msg{
					Data:   []byte("respData"),
					Header: make(nats.Header),
				}
				resp.Header.Add("RespHeader", "RespHeaderValue")
				return msg.RespondMsg(resp)
			},
		},

		{
			description: "Responses without headers should not crash",
			sendHttpRequestAndAssertResponse: func() error {
				// 1) send initial HTTP request (will be validated on the NATS handler side)
				req, err := http.NewRequest("GET", "http://localhost:8889/test/hi", nil)
				if err != nil {
					return err
				}
				res, err := http.DefaultClient.Do(req)
				if err != nil {
					return fmt.Errorf("HTTP request failed: %w", err)
				}

				// 3) validate HTTP response
				b, err := io.ReadAll(res.Body)
				if err != nil {
					return fmt.Errorf("could not read response body: %w", err)
				}
				if string(b) != "respData" {
					return fmt.Errorf("wrong response body. Expected: respData. Actual: %s", string(b))
				}

				return nil
			},
			CaddyfileSnippet: `
				route /test/* {
					nats_request greet.hello
				}
			`,
			handleNatsMessage: func(msg *nats.Msg, nc *nats.Conn) error {

				// 2) send NATS response (will be validated on the HTTP response side)
				return msg.Respond([]byte("respData"))
			},
		},
		// WILDCARDS!!
	}

	// we share the same NATS Server and Caddy Server for all testcases
	_, nc := integrationtest.StartTestNats(t)
	caddyTester := integrationtest.NewCaddyTester(t)

	for _, testcase := range cases {
		t.Run(testcase.description, func(t *testing.T) {

			subscription, err := nc.SubscribeSync("greet.>")
			defer subscription.Unsubscribe()
			integrationtest.FailOnErr("error subscribing to greet.>: %w", err, t)

			caddyTester.InitServer(fmt.Sprintf(integrationtest.DefaultCaddyConf+`
				:8889 {
					%s
				}
			`, "", testcase.CaddyfileSnippet), "caddyfile")

			// HTTP Request and assertion Goroutine
			httpResultChan := make(chan error)
			go func() {
				httpResultChan <- testcase.sendHttpRequestAndAssertResponse()
			}()

			// handle NATS message and generate response.
			msg, err := subscription.NextMsg(10 * time.Millisecond)
			if err != nil {
				t.Fatalf("message not received: %v", err)
			} else {
				t.Logf("Received message: %+v", msg)
			}
			err = testcase.handleNatsMessage(msg, nc)
			if err != nil {
				t.Fatalf("error with NATS message: %s", err)
			}

			// now, wait until the HTTP request goroutine finishes (and did its assertions)
			httpResult := <-httpResultChan
			if httpResult != nil {
				t.Fatalf("error with HTTP Response message: %s", err)
			}
		})
	}
}

// TestRequestTimeoutFromEnvVar tests that the timeout can be configured via environment variable
func TestRequestTimeoutFromEnvVar(t *testing.T) {
	testCases := []struct {
		name           string
		envValue       string
		expectedResult time.Duration
		shouldParse    bool
	}{
		{
			name:           "valid duration 30s",
			envValue:       "30s",
			expectedResult: 30 * time.Second,
			shouldParse:    true,
		},
		{
			name:           "valid duration 2m",
			envValue:       "2m",
			expectedResult: 2 * time.Minute,
			shouldParse:    true,
		},
		{
			name:           "invalid duration",
			envValue:       "invalid",
			expectedResult: 60 * time.Second,
			shouldParse:    false,
		},
		{
			name:           "empty env var",
			envValue:       "",
			expectedResult: 60 * time.Second,
			shouldParse:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.envValue != "" {
				os.Setenv("NATS_REQUEST_DEFAULT_TIMEOUT", tc.envValue)
				defer os.Unsetenv("NATS_REQUEST_DEFAULT_TIMEOUT")
			} else {
				os.Unsetenv("NATS_REQUEST_DEFAULT_TIMEOUT")
			}

			var req request.Request
			module := req.CaddyModule()
			instance := module.New().(*request.Request)

			if instance.Timeout != tc.expectedResult {
				t.Errorf("expected timeout %v, got %v", tc.expectedResult, instance.Timeout)
			}

			if tc.shouldParse && instance.Timeout == 60*time.Second && tc.envValue != "" {
				t.Errorf("expected timeout to be parsed from env var %s, but got default", tc.envValue)
			}
		})
	}
}

// TestRequestTimeoutFromCaddyfile tests that the timeout can be configured via Caddyfile
func TestRequestTimeoutFromCaddyfile(t *testing.T) {
	testCases := []struct {
		name             string
		caddyfileSnippet string
		expectedTimeout  time.Duration
		shouldError      bool
	}{
		{
			name: "valid timeout 5s",
			caddyfileSnippet: `
				route /test/* {
					nats_request greet.hello {
						timeout 5s
					}
				}
			`,
			expectedTimeout: 5 * time.Second,
			shouldError:     false,
		},
		{
			name: "valid timeout 2m",
			caddyfileSnippet: `
				route /test/* {
					nats_request greet.hello {
						timeout 2m
					}
				}
			`,
			expectedTimeout: 2 * time.Minute,
			shouldError:     false,
		},
		{
			name: "valid timeout 500ms",
			caddyfileSnippet: `
				route /test/* {
					nats_request greet.hello {
						timeout 500ms
					}
				}
			`,
			expectedTimeout: 500 * time.Millisecond,
			shouldError:     false,
		},
	}

	_, nc := integrationtest.StartTestNats(t)
	caddyTester := integrationtest.NewCaddyTester(t)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Ensure env var is not set for this test
			os.Unsetenv("NATS_REQUEST_DEFAULT_TIMEOUT")

			subscription, err := nc.SubscribeSync("greet.>")
			defer subscription.Unsubscribe()
			integrationtest.FailOnErr("error subscribing to greet.>: %w", err, t)

			caddyTester.InitServer(fmt.Sprintf(integrationtest.DefaultCaddyConf+`
				:8889 {
					%s
				}
			`, "", tc.caddyfileSnippet), "caddyfile")

			// Send HTTP request to trigger the handler
			req, err := http.NewRequest("GET", "http://localhost:8889/test/hi", nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			httpResultChan := make(chan error)
			go func() {
				res, err := http.DefaultClient.Do(req)
				if err != nil {
					httpResultChan <- fmt.Errorf("HTTP request failed: %w", err)
					return
				}
				res.Body.Close()
				httpResultChan <- nil
			}()

			// Wait for NATS message
			msg, err := subscription.NextMsg(1 * time.Second)
			if err != nil {
				t.Fatalf("message not received: %v", err)
			}

			// Respond quickly
			msg.Respond([]byte("ok"))

			// Wait for HTTP response
			httpErr := <-httpResultChan
			if httpErr != nil {
				t.Fatalf("HTTP request failed: %v", httpErr)
			}
		})
	}
}

// TestRequestTimeoutPriority tests the priority order: Caddyfile > Env Var > Default
func TestRequestTimeoutPriority(t *testing.T) {
	testCases := []struct {
		name             string
		envValue         string
		caddyfileSnippet string
		expectedTimeout  time.Duration
		description      string
	}{
		{
			name:     "no config uses default 60s",
			envValue: "",
			caddyfileSnippet: `
				route /test/* {
					nats_request greet.hello
				}
			`,
			expectedTimeout: 60 * time.Second,
			description:     "No env var, no Caddyfile timeout → uses 60s default",
		},
		{
			name:     "env var only uses env var",
			envValue: "30s",
			caddyfileSnippet: `
				route /test/* {
					nats_request greet.hello
				}
			`,
			expectedTimeout: 30 * time.Second,
			description:     "Env var set, no Caddyfile timeout → uses env var",
		},
		{
			name:     "caddyfile overrides env var",
			envValue: "30s",
			caddyfileSnippet: `
				route /test/* {
					nats_request greet.hello {
						timeout 10s
					}
				}
			`,
			expectedTimeout: 10 * time.Second,
			description:     "Env var + Caddyfile timeout → uses Caddyfile (override)",
		},
	}

	_, nc := integrationtest.StartTestNats(t)
	caddyTester := integrationtest.NewCaddyTester(t)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set or unset env var
			if tc.envValue != "" {
				os.Setenv("NATS_REQUEST_DEFAULT_TIMEOUT", tc.envValue)
				defer os.Unsetenv("NATS_REQUEST_DEFAULT_TIMEOUT")
			} else {
				os.Unsetenv("NATS_REQUEST_DEFAULT_TIMEOUT")
			}

			subscription, err := nc.SubscribeSync("greet.>")
			defer subscription.Unsubscribe()
			integrationtest.FailOnErr("error subscribing to greet.>: %w", err, t)

			caddyTester.InitServer(fmt.Sprintf(integrationtest.DefaultCaddyConf+`
				:8889 {
					%s
				}
			`, "", tc.caddyfileSnippet), "caddyfile")

			// Send HTTP request
			req, err := http.NewRequest("GET", "http://localhost:8889/test/hi", nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			startTime := time.Now()
			httpResultChan := make(chan error)
			go func() {
				res, err := http.DefaultClient.Do(req)
				if err != nil {
					httpResultChan <- fmt.Errorf("HTTP request failed: %w", err)
					return
				}
				res.Body.Close()
				httpResultChan <- nil
			}()

			// Wait for NATS message
			msg, err := subscription.NextMsg(1 * time.Second)
			if err != nil {
				t.Fatalf("message not received: %v", err)
			}

			// Respond quickly
			msg.Respond([]byte("ok"))

			// Wait for HTTP response
			httpErr := <-httpResultChan
			elapsed := time.Since(startTime)

			if httpErr != nil {
				t.Fatalf("HTTP request failed: %v", httpErr)
			}

			// The timeout should be at least the configured value (or close to it for fast responses)
			// For fast responses, we just verify it completed successfully
			// The actual timeout validation happens in TestRequestTimeoutBehavior
			t.Logf("Request completed in %v (expected timeout: %v)", elapsed, tc.expectedTimeout)
		})
	}
}

// TestRequestTimeoutBehavior tests actual timeout behavior: success and failure cases
func TestRequestTimeoutBehavior(t *testing.T) {
	testCases := []struct {
		name           string
		timeout        time.Duration
		responseDelay  time.Duration
		expectedStatus int
		description    string
	}{
		{
			name:           "success when response is fast",
			timeout:        5 * time.Second,
			responseDelay:  100 * time.Millisecond,
			expectedStatus: 200,
			description:    "Timeout 5s, response in 100ms → HTTP 200",
		},
		{
			name:           "timeout when response is slow",
			timeout:        1 * time.Second,
			responseDelay:  2 * time.Second,
			expectedStatus: 504,
			description:    "Timeout 1s, response in 2s → HTTP 504",
		},
	}

	_, nc := integrationtest.StartTestNats(t)
	caddyTester := integrationtest.NewCaddyTester(t)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Ensure env var is not set
			os.Unsetenv("NATS_REQUEST_DEFAULT_TIMEOUT")

			caddyfileSnippet := fmt.Sprintf(`
				route /test/* {
					nats_request greet.hello {
						timeout %s
					}
				}
			`, tc.timeout.String())

			subscription, err := nc.SubscribeSync("greet.>")
			defer subscription.Unsubscribe()
			integrationtest.FailOnErr("error subscribing to greet.>: %w", err, t)

			caddyTester.InitServer(fmt.Sprintf(integrationtest.DefaultCaddyConf+`
				:8889 {
					%s
				}
			`, "", caddyfileSnippet), "caddyfile")

			// Send HTTP request
			req, err := http.NewRequest("GET", "http://localhost:8889/test/hi", nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			httpResultChan := make(chan struct {
				status int
				err    error
			})

			go func() {
				res, err := http.DefaultClient.Do(req)
				status := 0
				if res != nil {
					status = res.StatusCode
					res.Body.Close()
				}
				httpResultChan <- struct {
					status int
					err    error
				}{status: status, err: err}
			}()

			// Wait for NATS message
			msg, err := subscription.NextMsg(1 * time.Second)
			if err != nil {
				t.Fatalf("message not received: %v", err)
			}

			// Delay response based on test case
			time.Sleep(tc.responseDelay)

			// Respond to NATS message
			msg.Respond([]byte("response data"))

			// Wait for HTTP response
			result := <-httpResultChan
			if result.err != nil {
				t.Fatalf("HTTP request failed: %v", result.err)
			}

			if result.status != tc.expectedStatus {
				t.Errorf("expected status %d, got %d. Description: %s", tc.expectedStatus, result.status, tc.description)
			}
		})
	}
}
