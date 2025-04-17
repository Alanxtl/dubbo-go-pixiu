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
	"context"
	"fmt"
	"net"
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
	_ "github.com/apache/dubbo-go-pixiu/pkg/common/extension/filter"
	"github.com/apache/dubbo-go-pixiu/pkg/common/mock"
	"github.com/apache/dubbo-go-pixiu/pkg/common/router/trie"
	contexthttp "github.com/apache/dubbo-go-pixiu/pkg/context/http"
	"github.com/apache/dubbo-go-pixiu/pkg/logger"
	"github.com/apache/dubbo-go-pixiu/pkg/model"
)

var (
	eventCh = make(chan string, 3)
)

// test SSE case
func TestStreamingResponse(t *testing.T) {
	hcmc := model.HttpConnectionManagerConfig{
		RouteConfig: model.RouteConfiguration{
			RouteTrie: trie.NewTrieWithDefault("GET/api/sse", model.RouteAction{
				Cluster: "mock_stream_cluster",
			}),
		},
		HTTPFilters: []*model.HTTPFilter{
			{
				Name: mock.Kind,
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// mock server
	upstreamServer, _ := NewTestServerWithURL("localhost:8080", stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		flusher := w.(stdhttp.Flusher)

		for i := 1; i <= 3; i++ {
			select {
			case <-ctx.Done():
				return
			default:
				time.Sleep(10 * time.Millisecond)
				event := fmt.Sprintf("data: %d\nevent: %d\nid: %d\n\n", i, i, i)
				_, _ = w.Write([]byte(event))
				flusher.Flush()
				logger.Info("Upstream sent event ", i)
			}
		}
	}))
	defer upstreamServer.Close()

	req := httptest.NewRequest("GET", "http://localhost:8080/api/sse", nil).WithContext(ctx)

	done := make(chan struct{})

	httpCtx := &contexthttp.HttpContext{
		Request: req,
		Writer:  NewStreamRecorder(),
		Ctx:     ctx,
	}
	go func() {
		defer close(done)

		hcm := CreateHttpConnectionManager(&hcmc)

		if err := hcm.Handle(httpCtx); err != nil {
			t.Errorf("Handle failed: %v", err)
		}

		// test targetResp
		if httpCtx.TargetResp == nil {
			t.Error("TargetResp is nil")
			return
		}
	}()

	// event waiting test
	for {
		receivedEvents := httpCtx.Writer.(*StreamRecorder).receivedBuf
		select {
		case event := <-eventCh:
			logger.Info("Received event: %s", strings.ReplaceAll(event, "\n", "\\n"))
		case <-done:
			assert.Equal(t, 3, len(receivedEvents), "Should receive 3 events")
			return
		case <-time.After(5 * time.Second):
			t.Fatal("Test timeout")
			return
		}
	}

}

// mock recorder
type StreamRecorder struct {
	stdhttp.ResponseWriter
	stdhttp.Flusher
	receivedBuf []string
	headers     stdhttp.Header
	status      int
}

func NewStreamRecorder() *StreamRecorder {
	return &StreamRecorder{
		receivedBuf: make([]string, 0),
		headers:     make(stdhttp.Header),
	}
}

func (r *StreamRecorder) Header() stdhttp.Header {
	return r.headers
}

func (r *StreamRecorder) WriteHeader(statusCode int) {
	r.status = statusCode
}

func (r *StreamRecorder) Write(data []byte) (int, error) {
	eventCh <- string(data)
	r.receivedBuf = append(r.receivedBuf, string(data))
	return len(data), nil
}

func NewTestServerWithURL(URL string, handler stdhttp.Handler) (*httptest.Server, error) {
	ts := httptest.NewUnstartedServer(handler)
	if URL != "" {
		l, err := net.Listen("tcp", URL)
		if err != nil {
			return nil, err
		}
		ts.Listener.Close()
		ts.Listener = l
	}
	ts.Start()
	return ts, nil
}
