package request_test

import (
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	_ "github.com/buarki/caddy-nats-bridge"
	"github.com/buarki/caddy-nats-bridge/integrationtest"
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

		{
			description: "Global default timeout should be used when no route-level timeout is specified",
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

				// 3) validate HTTP response - should get a timeout error (504 Gateway Timeout)
				if res.StatusCode != http.StatusGatewayTimeout {
					return fmt.Errorf("expected HTTP status 504 (Gateway Timeout), got %d", res.StatusCode)
				}

				return nil
			},
			CaddyfileSnippet: `
				route /test/* {
					nats_request greet.hello
				}
			`,
			handleNatsMessage: func(msg *nats.Msg, nc *nats.Conn) error {
				// 2) simulate a slow NATS response that exceeds the global timeout (5s)
				// This should cause the request to timeout
				time.Sleep(6 * time.Second)
				return msg.Respond([]byte("respData"))
			},
		},

		{
			description: "Route-level timeout should override global default timeout",
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

				// 3) validate HTTP response - should get a timeout error (504 Gateway Timeout)
				// because the route-level timeout (1s) is shorter than the NATS response delay (2s)
				if res.StatusCode != http.StatusGatewayTimeout {
					return fmt.Errorf("expected HTTP status 504 (Gateway Timeout), got %d", res.StatusCode)
				}

				return nil
			},
			CaddyfileSnippet: `
				route /test/* {
					nats_request greet.hello {
						timeout 1s
					}
				}
			`,
			handleNatsMessage: func(msg *nats.Msg, nc *nats.Conn) error {
				// 2) simulate a slow NATS response that exceeds the route-level timeout (1s)
				// but is shorter than the global timeout (5s)
				time.Sleep(2 * time.Second)
				return msg.Respond([]byte("respData"))
			},
		},

		{
			description: "Request should succeed when NATS response is within global default timeout",
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

				// 3) validate HTTP response - should succeed (200 OK)
				if res.StatusCode != http.StatusOK {
					return fmt.Errorf("expected HTTP status 200 (OK), got %d", res.StatusCode)
				}

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
				// 2) simulate a fast NATS response that is within the global timeout (5s)
				time.Sleep(1 * time.Second)
				return msg.Respond([]byte("respData"))
			},
		},

		{
			description: "Route-level timeout should work when no global default is set",
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

				// 3) validate HTTP response - should get a timeout error (504 Gateway Timeout)
				// because the route-level timeout (1s) is shorter than the NATS response delay (2s)
				if res.StatusCode != http.StatusGatewayTimeout {
					return fmt.Errorf("expected HTTP status 504 (Gateway Timeout), got %d", res.StatusCode)
				}

				return nil
			},
			CaddyfileSnippet: `
				route /test/* {
					nats_request greet.hello {
						timeout 1s
					}
				}
			`,
			handleNatsMessage: func(msg *nats.Msg, nc *nats.Conn) error {
				// 2) simulate a slow NATS response that exceeds the route-level timeout (1s)
				time.Sleep(2 * time.Second)
				return msg.Respond([]byte("respData"))
			},
		},

		{
			description: "Package default timeout should be used when neither global nor route-level is set",
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

				// 3) validate HTTP response - should get a timeout error (504 Gateway Timeout)
				// because the package default timeout (60s) is shorter than the NATS response delay (61s)
				if res.StatusCode != http.StatusGatewayTimeout {
					return fmt.Errorf("expected HTTP status 504 (Gateway Timeout), got %d", res.StatusCode)
				}

				return nil
			},
			CaddyfileSnippet: `
				route /test/* {
					nats_request greet.hello
				}
			`,
			handleNatsMessage: func(msg *nats.Msg, nc *nats.Conn) error {
				// 2) simulate a slow NATS response that exceeds the package default timeout (60s)
				time.Sleep(61 * time.Second)
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

			// Use different Caddy configuration for the global timeout test
			var caddyConfig string
			if testcase.description == "Global default timeout should be used when no route-level timeout is specified" ||
				testcase.description == "Route-level timeout should override global default timeout" ||
				testcase.description == "Request should succeed when NATS response is within global default timeout" {
				// Configure NATS with global default timeout
				caddyConfig = fmt.Sprintf(`
					{
						default_bind 127.0.0.1
						http_port 8889
						admin 127.0.0.1:2999
						nats {
							url 127.0.0.1:8369
							defaultTimeout 5s
						}
					}
					:8889 {
						%s
					}
				`, testcase.CaddyfileSnippet)
			} else {
				// Use default configuration for other tests (no global timeout set)
				caddyConfig = fmt.Sprintf(integrationtest.DefaultCaddyConf+`
					:8889 {
						%s
					}
				`, "", testcase.CaddyfileSnippet)
			}

			caddyTester.InitServer(caddyConfig, "caddyfile")

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
				t.Fatalf("error with HTTP Response message: %s", httpResult)
			}
		})
	}
}
