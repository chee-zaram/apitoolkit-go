package apitoolkit

import (
	"bytes"
	"context"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

type roundTripper struct {
	base   http.RoundTripper
	ctx    context.Context
	client *Client
	cfg  *roundTripperConfig
}

func (rt *roundTripper) RoundTrip(req *http.Request) (res *http.Response, err error) {
	defer func() {
		// span.Finish(tracer.WithError(err))
	}()

	if rt.client == nil {
		log.Println("APIToolkit: outgoing rountripper has a nil Apitoolkit client.")
		return rt.base.RoundTrip(req)
	}

	// Capture the request body
	reqBodyBytes := []byte{}
	if req.Body != nil {
		reqBodyBytes, _ = ioutil.ReadAll(req.Body)
		req.Body = ioutil.NopCloser(bytes.NewBuffer(reqBodyBytes))
	}

	// Add a header to all outgoing requests "X-APITOOLKIT-TRACE-PARENT-ID"
	start := time.Now()
	res, err = rt.base.RoundTrip(req)

	// Capture the response body
	respBodyBytes, _ := ioutil.ReadAll(res.Body)
	res.Body = ioutil.NopCloser(bytes.NewBuffer(respBodyBytes))

	payload := rt.client.buildPayload(
		GoOutgoing,
		start, req, res.StatusCode, reqBodyBytes,
		respBodyBytes, res.Header, nil,
		req.URL.Path, 
		rt.cfg.RedactHeaders, rt.cfg.RedactRequestBody, rt.cfg.RedactResponseBody,
	)

	pErr := rt.client.PublishMessage(req.Context(), payload)
	if pErr != nil {
		if rt.client.config.Debug {
			log.Println("APIToolkit: unable to publish outgoing request payload to pubsub.")
		}
	}
	return res, err
}

type roundTripperConfig struct {
	RedactHeaders      []string
	RedactRequestBody  []string
	RedactResponseBody []string
}

type RoundTripperOption func(*roundTripperConfig)

func WithRedactHeaders(headers []string) RoundTripperOption {
	return func(rc *roundTripperConfig) {
		rc.RedactHeaders = headers
	}
}

func WithRedactRequestBody(fields []string) RoundTripperOption {
	return func(rc *roundTripperConfig) {
		rc.RedactRequestBody = fields
	}
}

func WithRedactResponseBody(fields []string) RoundTripperOption {
	return func(rc *roundTripperConfig) {
		rc.RedactResponseBody = fields
	}
}

// WrapRoundTripper returns a new RoundTripper which traces all requests sent
// over the transport.
func (c *Client) WrapRoundTripper(ctx context.Context, rt http.RoundTripper, opts ...RoundTripperOption) http.RoundTripper {
	cfg := new(roundTripperConfig)
	for _, opt := range opts {
		opt(cfg)
	}

	// If no rt is passed in, then use the default standard library transport
	if rt == nil {
		rt = http.DefaultTransport
	}
	return &roundTripper{
		base:   rt,
		ctx:    ctx,
		client: c,
		cfg: cfg,
	}
}
