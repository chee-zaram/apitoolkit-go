package apitoolkit

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/imroc/req"
	"github.com/kr/pretty"
	"github.com/stretchr/testify/assert"
)

func TestErrorReporting(t *testing.T) {
	client := &Client{
		config: &Config{
			RedactHeaders:      []string{"X-Api-Key", "Accept-Encoding"},
			RedactResponseBody: exampleDataRedaction,
		},
	}
	var publishCalled bool
	client.PublishMessage = func(ctx context.Context, payload Payload) error {
		
		x, _ := json.MarshalIndent(payload, "", "\t")
		fmt.Println(string(x))
		pretty.Println(payload.Errors)

		assert.Equal(t, "POST", payload.Method)
		assert.Equal(t, "/test", payload.URLPath)
		publishCalled = true
		return nil
	}

	handlerFn := func(w http.ResponseWriter, r *http.Request) {
		err1 := errors.Newf("Example Error %v", "value")

		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"key":"value"}`))
		err2 := errors.Wrap(err1, "wrapper from err2")
		ReportError(r.Context(), err2)
	}

	ts := httptest.NewServer(client.Middleware(http.HandlerFunc(handlerFn)))
	defer ts.Close()

	outClient := &Client{
		config: &Config{},
	}

	outClient.PublishMessage = func(ctx context.Context, payload Payload) error {
		assert.Equal(t, "/test?param1=abc&param2=123", payload.RawURL)
		assert.Equal(t, http.StatusAccepted, payload.StatusCode)
		assert.Greater(t, payload.Duration, 1000*time.Nanosecond)
		assert.Equal(t, GoOutgoing, payload.SdkType)
		return nil
	}
	ctx := context.Background()
	atHTTPClient := http.DefaultClient
	atHTTPClient.Transport = outClient.WrapRoundTripper(
		ctx, atHTTPClient.Transport,
		WithRedactHeaders([]string{}),
	)
	req.SetClient(atHTTPClient)
	_, err := req.Post(ts.URL+"/test",
		req.Param{"param1": "abc", "param2": 123},
		req.Header{
			"Content-Type": "application/json",
			"X-API-KEY":    "past-3",
		},
		req.BodyJSON(exampleData2),
	)
	assert.NoError(t, err)
	assert.True(t, publishCalled)
}