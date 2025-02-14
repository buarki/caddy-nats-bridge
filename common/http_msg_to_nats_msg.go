package common

import (
	"fmt"
	"io"
	"net/http"

	"github.com/nats-io/nats.go"
)

var subjectRgx = `[\>\$\s]`

var invalidChars = map[rune]*struct{}{
	'>': nil,
	'$': nil,
	' ': nil,
}

// NatsMsgForHttpRequest creates a nats.Msg from an existing http.Request: the HTTP Request Body is transferred
// to the NATS message Data, and the headers are transferred as well.
//
// Three special headers are added for the request method, URL path, and raw query.
func NatsMsgForHttpRequest(r *http.Request, subject string) (*nats.Msg, error) {
	var msg *nats.Msg
	b, _ := io.ReadAll(r.Body)

	headers := nats.Header(r.Header)
	for k, v := range ExtraNatsMsgHeadersFromContext(r.Context()) {
		headers.Add(k, v)
	}

	for _, v := range subject {
		if _, ok := invalidChars[v]; ok {
			return nil, fmt.Errorf("invalid character in subject %v", subject)
		}
	}

	msg = &nats.Msg{
		Subject: subject,
		Header:  headers,
		Data:    b,
	}

	msg.Header.Add("X-NatsBridge-Method", r.Method)
	msg.Header.Add("X-NatsBridge-UrlPath", r.URL.Path)
	msg.Header.Add("X-NatsBridge-UrlQuery", r.URL.RawQuery)
	//if err := queryToHeaders(r.URL.RawQuery, msg); err != nil {
	//	return nil, err
	//}

	return msg, nil
}

//func queryToHeaders(query string, msg *nats.Msg) error {
//	q, err := url.ParseQuery(query)
//	if err != nil {
//		return err
//	}
//
//	var headers map[string][]string
//	for k, v := range q {
//		if strings.ToLower(k) == "authorization" {
//			continue
//		}
//		if strings.ToLower(k) == "x-request-id" {
//			continue
//		}
//		msg.Header.Add(k, v)
//	}
//
//	msg.Header = append(msg.Header, headers)
//
//	return
//}
