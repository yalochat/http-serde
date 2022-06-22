package http_serde

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/yalochat/http-serde/internal/mocks"
)

func TestNew(t *testing.T) {
	tests := []struct {
		it     string
		assert func(t *testing.T, got interface{})
	}{
		{
			it: "returns a new http request de/serializer",
			assert: func(t *testing.T, got interface{}) {
				_, ok := got.(SerDe)
				require.True(t, ok)
				_, ok = got.(*serde)
				require.True(t, ok)
			},
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.it, func(t *testing.T) {
				tt.assert(t, New())
			},
		)
	}
}

func TestSerialize(t *testing.T) {
	tests := []struct {
		it     string
		setup  func(t *testing.T) *http.Request
		assert func(t *testing.T, b []byte, err error)
	}{
		{
			it: "returns an error if http request is nil",
			setup: func(t *testing.T) *http.Request {
				return nil
			},
			assert: func(t *testing.T, b []byte, err error) {
				require.Error(t, err)
				require.Nil(t, b)
			},
		},
		{
			it: "returns an error if http request body cannot be read",
			setup: func(t *testing.T) *http.Request {
				body := &mocks.FakeReadCloser{}
				body.ReadReturns(0, errors.New("test"))
				body.CloseReturns(nil)
				return &http.Request{Body: body}
			},
			assert: func(t *testing.T, b []byte, err error) {
				require.Error(t, err)
				require.Nil(t, b)
				require.Equal(t, "test", err.Error())
			},
		},
		{
			it: "returns an error if http request body cannot be closed",
			setup: func(t *testing.T) *http.Request {
				body := &mocks.FakeReadCloser{}
				body.ReadReturns(0, io.EOF)
				body.CloseReturns(errors.New("test"))
				return &http.Request{Body: body}
			},
			assert: func(t *testing.T, b []byte, err error) {
				require.Error(t, err)
				require.Nil(t, b)
				require.Equal(t, "test", err.Error())
			},
		},
		{
			it: "serializes GET requests",
			setup: func(t *testing.T) *http.Request {
				req, err := http.NewRequest(http.MethodGet, "http://test.test/test", nil)
				require.NoError(t, err)
				return req
			},
			assert: func(t *testing.T, b []byte, err error) {
				require.NoError(t, err)
				require.NotNil(t, b)
				require.Equal(
					t, strings.Join(
						[]string{
							"GET /test HTTP/1.1",
							"Host: test.test",
							"Content-Length: 0",
							"",
							"",
						}, "\r\n",
					), string(b),
				)
			},
		},
		{
			it: "serializes POST requests",
			setup: func(t *testing.T) *http.Request {
				requestBody := io.NopCloser(bytes.NewBuffer([]byte("test")))
				req, err := http.NewRequest(http.MethodPost, "http://test.test", requestBody)
				require.NoError(t, err)
				return req
			},
			assert: func(t *testing.T, b []byte, err error) {
				require.NoError(t, err)
				require.NotNil(t, b)
				require.Equal(
					t, strings.Join(
						[]string{
							"POST / HTTP/1.1",
							"Host: test.test",
							"Content-Length: 4",
							"",
							"test",
						}, "\r\n",
					), string(b),
				)
			},
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.it, func(t *testing.T) {
				req := tt.setup(t)
				got, err := New().Serialize(req)
				tt.assert(t, got, err)
			},
		)
	}
}

func TestDeserialize(t *testing.T) {
	tests := []struct {
		it     string
		setup  func(t *testing.T) []byte
		assert func(t *testing.T, req *http.Request, err error)
	}{
		{
			it: "returns an error if serialized request is invalid",
			setup: func(t *testing.T) []byte {
				return []byte("INVALID")
			},
			assert: func(t *testing.T, req *http.Request, err error) {
				require.Error(t, err)
				require.Nil(t, req)
			},
		},
		{
			it: "deserializes GET requests",
			setup: func(t *testing.T) []byte {
				req, err := http.NewRequest(http.MethodGet, "http://test.test/test?foo=bar", nil)
				require.NoError(t, err)
				ser, err := httputil.DumpRequest(req, true)
				require.NoError(t, err)
				return ser
			},
			assert: func(t *testing.T, req *http.Request, err error) {
				require.NoError(t, err)
				require.NotNil(t, req)
				require.Equal(t, http.MethodGet, req.Method)
				require.Equal(t, "test.test", req.Host)
				require.Equal(t, "/test", req.URL.Path)
				require.Equal(t, "bar", req.URL.Query().Get("foo"))
			},
		},
		{
			it: "deserializes POST requests",
			setup: func(t *testing.T) []byte {
				requestBody := io.NopCloser(bytes.NewBuffer([]byte("test")))
				req, err := http.NewRequest(http.MethodPost, "http://test.test", requestBody)
				req.Header.Set("Content-Length", "4")
				require.NoError(t, err)
				ser, err := httputil.DumpRequest(req, true)
				require.NoError(t, err)
				return ser
			},
			assert: func(t *testing.T, req *http.Request, err error) {
				require.NoError(t, err)
				require.NotNil(t, req)
				require.Equal(t, http.MethodPost, req.Method)
				require.Equal(t, "test.test", req.Host)
				require.NotNil(t, req.Body)
				b, err := ioutil.ReadAll(req.Body)
				require.NoError(t, err)
				require.Equal(t, "test", string(b))
			},
		},
		{
			it: "does not deserialize bodies when content length header is not present",
			setup: func(t *testing.T) []byte {
				requestBody := io.NopCloser(bytes.NewBuffer([]byte("test")))
				req, err := http.NewRequest(http.MethodPost, "http://test.test", requestBody)
				require.NoError(t, err)
				ser, err := httputil.DumpRequest(req, true)
				require.NoError(t, err)
				return ser
			},
			assert: func(t *testing.T, req *http.Request, err error) {
				require.NoError(t, err)
				require.NotNil(t, req)
				require.Equal(t, http.MethodPost, req.Method)
				require.Equal(t, "test.test", req.Host)
				require.NotNil(t, req.Body)
				b, err := ioutil.ReadAll(req.Body)
				require.NoError(t, err)
				require.Equal(t, "", string(b))
			},
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.it, func(t *testing.T) {
				s := tt.setup(t)
				got, err := New().Deserialize(s)
				tt.assert(t, got, err)
			},
		)
	}
}

func TestGinIntegration(t *testing.T) {
	tests := []struct {
		it     string
		setup  func(t *testing.T, ctx context.Context, out chan<- []byte) (http.Handler, *http.Request)
		assert func(t *testing.T, r *http.Request, err error)
	}{
		{
			it: "serializes incoming request from gin handler",
			setup: func(t *testing.T, ctx context.Context, out chan<- []byte) (http.Handler, *http.Request) {
				router := gin.Default()
				router.GET(
					"/serialize", func(ctx *gin.Context) {
						got, err := New().Serialize(ctx.Request)
						require.NoError(t, err)
						require.NotNil(t, got)
						out <- got
					},
				)
				request, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost:8080/serialize", nil)
				return router, request
			},
			assert: func(t *testing.T, r *http.Request, err error) {

			},
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.it, func(t *testing.T) {
				serDe := New()
				ctx, cancelFunc := context.WithTimeout(context.Background(), time.Hour)
				defer cancelFunc()
				out := make(chan []byte, 1)
				handler, request := tt.setup(t, ctx, out)
				s := http.Server{
					Addr:    ":8080",
					Handler: handler,
				}
				go func() {
					err := s.ListenAndServe()
					require.True(t, err != nil && errors.Is(err, http.ErrServerClosed), "got error", err)
				}()

				response, err := http.DefaultClient.Do(request)
				require.NoError(t, err)
				require.Equal(t, http.StatusOK, response.StatusCode)

				deserialized, err := serDe.Deserialize(<-out)
				require.NoError(t, err)

				// Request URL contains the path within the instance.
				deserialized.URL, err = url.Parse("http://localhost" + s.Addr + deserialized.RequestURI)

				// Request URI must be empty in order to be executed, else we get
				// **http: Request.RequestURI can't be set in client requests** error
				deserialized.RequestURI = ""

				require.NoError(t, err)
				response, err = http.DefaultClient.Do(deserialized)
				require.NoError(t, err)
				require.Equal(t, http.StatusOK, response.StatusCode)
				require.NoError(t, s.Shutdown(ctx))
				deserialized2, err := serDe.Deserialize(<-out)
				require.NoError(t, err)

				// Since both URLs are different memory addresses, direct comparison doesn't work.
				b1, _ := serDe.Serialize(deserialized)
				b2, _ := serDe.Serialize(deserialized2)
				require.Equal(t, b1, b2)
			},
		)
	}
}
