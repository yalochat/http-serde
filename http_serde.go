package http_serde

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httputil"
	"strconv"
)

type Serializer interface {
	Serialize(request *http.Request) ([]byte, error)
}

type Deserializer interface {
	Deserialize(serialized []byte) (*http.Request, error)
}

type SerDe interface {
	Serializer
	Deserializer
}

type serde struct{}

func contentLength(request *http.Request) (int, error) {
	if request.Body == nil || request.Body == http.NoBody {
		return 0, nil
	}
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(request.Body); err != nil {
		return 0, err
	}
	if err := request.Body.Close(); err != nil {
		return 0, err
	}
	request.Body = io.NopCloser(bytes.NewReader(buf.Bytes()))
	return buf.Len(), nil
}

func (s *serde) Serialize(request *http.Request) ([]byte, error) {
	if request == nil {
		return nil, errors.New("serialize called on nil request")
	}
	l, err := contentLength(request)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Length", strconv.Itoa(l))
	return httputil.DumpRequest(request, true)
}

func (s *serde) Deserialize(serialized []byte) (*http.Request, error) {
	req, err := http.ReadRequest(bufio.NewReader(bytes.NewBuffer(serialized)))
	if err != nil {
		return nil, err
	}
	return req, nil
}

func New() SerDe {
	return &serde{}
}
