/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package http

import (
	"fmt"
	"io"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go-pixiu/pkg/client/http"
	clienthttp "github.com/apache/dubbo-go-pixiu/pkg/client/http"
	"github.com/apache/dubbo-go-pixiu/pkg/common/extension/filter"
	"github.com/apache/dubbo-go-pixiu/pkg/common/router/trie"
	contexthttp "github.com/apache/dubbo-go-pixiu/pkg/context/http"
	"github.com/apache/dubbo-go-pixiu/pkg/context/mock"
	"github.com/apache/dubbo-go-pixiu/pkg/logger"
	"github.com/apache/dubbo-go-pixiu/pkg/model"
)

var (
	SseDemo = "dgp.sse.filters.http.demo"
	// Kind is the kind of plugin.
	SseKind    = SseDemo
	encodeCall = false
)

type (
	// Plugin is http filter plugin.
	SsePlugin struct {
	}
	// HeaderFilter is http filter instance
	SseFilterFactory struct {
		conf *SseConfig
	}
	// SseFilter is http filter instance
	SseFilter struct {
		str        string
		encodeCall bool // trace encode call
	}

	// SseConfig describe the SseConfig of ResponseFilter
	SseConfig struct {
		Foo string `json:"foo,omitempty" yaml:"foo,omitempty"`
		Bar string `json:"bar,omitempty" yaml:"bar,omitempty"`
	}
)

func (p *SsePlugin) Kind() string {
	return SseKind
}

func (p *SsePlugin) CreateFilterFactory() (filter.HttpFilterFactory, error) {
	return &SseFilterFactory{conf: &SseConfig{Foo: "default foo", Bar: "default bar"}}, nil
}

type sseReadCloser struct {
	*strings.Reader
}

func (s *sseReadCloser) Close() error {
	return nil
}

func mockSSEReadCloser() io.ReadCloser {
	data := "data: event1\n\n" +
		"event: update\n" +
		"id: 123\n" +
		"data: {\"status\":\"processing\"}\n\n"

	return &sseReadCloser{
		Reader: strings.NewReader(data),
	}
}
func (f *SseFilter) Decode(ctx *contexthttp.HttpContext) filter.FilterStatus {
	logger.Info("decode phase: ", f.str)

	runes := []rune(f.str)
	for i := 0; i < len(runes)/2; i += 1 {
		runes[i], runes[len(runes)-1-i] = runes[len(runes)-1-i], runes[i]
	}
	f.str = string(runes)

	mockResp := &stdhttp.Response{
		StatusCode: stdhttp.StatusOK,
		Header: stdhttp.Header{
			"Content-Type":  []string{"text/event-stream"},
			"Cache-Control": []string{"no-cache"},
		},
		Body: mockSSEReadCloser(),
	}

	ctx.SourceResp = clienthttp.NewSSEReader(mockResp.Body)
	return filter.Continue
}

func (f *SseFilter) Encode(ctx *contexthttp.HttpContext) filter.FilterStatus {
	encodeCall = true

	f.encodeCall = true
	logger.Info("encode phase: ", f.str)
	return filter.Continue
}

func (f *SseFilterFactory) PrepareFilterChain(ctx *contexthttp.HttpContext, chain filter.FilterChain) error {
	c := f.conf
	str := fmt.Sprintf("%s is drinking in the %s", c.Foo, c.Bar)
	SseFilter := &SseFilter{str: str}

	chain.AppendDecodeFilters(SseFilter)
	chain.AppendEncodeFilters(SseFilter)
	return nil
}

func (f *SseFilterFactory) Config() interface{} {
	return f.conf
}

func (f *SseFilterFactory) Apply() error {
	return nil
}

// test SSE case
func TestStreamingResponse(t *testing.T) {
	filter.RegisterHttpFilter(&SsePlugin{})
	hcmc := model.HttpConnectionManagerConfig{
		RouteConfig: model.RouteConfiguration{
			RouteTrie: trie.NewTrieWithDefault("GET/api/stream", model.RouteAction{
				Cluster: "test_stream",
			}),
		},
		HTTPFilters: []*model.HTTPFilter{
			{
				Name:   SseDemo,
				Config: nil,
			},
		},
	}

	hcm := CreateHttpConnectionManager(&hcmc)

	// mock SSE reader
	mockStream := &MockSSEReader{
		events: make(chan http.SSEEvent, 3),
		errCh:  make(chan error, 1),
	}

	// input test events
	mockStream.events <- http.SSEEvent{Data: []byte("event1")}
	mockStream.events <- http.SSEEvent{
		Event: "update",
		Data:  []byte(`{"status":"processing"}`),
		ID:    "123",
	}
	close(mockStream.events) // close the event channel to indicate stream end

	// create test request
	req, _ := stdhttp.NewRequest("GET", "http://example.com/api/stream", nil)
	ctx := mock.GetMockHTTPContext(req)

	// set stream response source
	ctx.SourceResp = mockStream

	// catch response
	recorder := httptest.NewRecorder()
	ctx.Writer = &BufferedResponseWriter{
		ResponseWriter: recorder,
		Flushed:        false,
	}

	// handle
	err := hcm.Handle(ctx)
	assert.NoError(t, err)

	logger.Info(recorder.Body)

	assert.Equal(t, stdhttp.StatusOK, recorder.Code)
	assert.Equal(t, "text/event-stream", recorder.Header().Get("Content-Type"))
	assert.Equal(t, "no-cache", recorder.Header().Get("Cache-Control"))

	time.Sleep(time.Second * 1)
	output := recorder.Body.String()

	expected := `data: event1


event: update
id: 123
data: {"status":"processing"}


`
	output = strings.Replace(output, "\n", "\\n", -1)
	expected = strings.Replace(expected, "\n", "\\n", -1)
	logger.Info("output:   ", output)
	logger.Info("expected: ", expected)
	assert.Contains(t, output, expected)
	// test filter behavior
	// test onencode not called

	assert.Equal(t, false, encodeCall)
}

// MockSSEReader mocks http.SSEReader for testing
type MockSSEReader struct {
	events chan http.SSEEvent
	errCh  chan error
}

func (m *MockSSEReader) Events() <-chan http.SSEEvent { return m.events }
func (m *MockSSEReader) Err() <-chan error            { return m.errCh }
func (m *MockSSEReader) Close() error                 { return nil }

// BufferedResponseWriter support Flusher test
type BufferedResponseWriter struct {
	stdhttp.ResponseWriter
	Flushed bool
}

func (b *BufferedResponseWriter) Flush() {
	if flusher, ok := b.ResponseWriter.(stdhttp.Flusher); ok {
		flusher.Flush()
		b.Flushed = true
	}
}

func (b *BufferedResponseWriter) Write(data []byte) (int, error) {
	return b.ResponseWriter.Write(data)
}
