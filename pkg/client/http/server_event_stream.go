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
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
)

import (
	"go.opentelemetry.io/otel/attribute"

	"go.opentelemetry.io/otel/trace"
)

import (
	"github.com/apache/dubbo-go-pixiu/pkg/common/constant"
)

// SSEEvent SSE Event after parsing
type SSEEvent struct {
	Event string
	Data  []byte
	ID    string
}

type SSEReader struct {
	body    io.ReadCloser
	scanner *bufio.Scanner
	eventCh chan SSEEvent
	errCh   chan error
}

// isSSEStream check if the response is a SSE stream
func isSSEStream(resp *http.Response) bool {
	contentType := resp.Header.Get(constant.HeaderKeyContextType)
	return contentType == constant.HeaderValueTextEventStream
}

func NewSSEReader(body io.ReadCloser) *SSEReader {
	s := &SSEReader{
		body:    body,
		scanner: bufio.NewScanner(body),
		eventCh: make(chan SSEEvent),
		errCh:   make(chan error, 1),
	}
	go s.parseStream()
	return s
}

// parseStream parses the SSE stream, refer to https://html.spec.whatwg.org/multipage/server-sent-events.html
func (s *SSEReader) parseStream() {
	defer close(s.eventCh)
	defer close(s.errCh)
	defer func(body io.ReadCloser) {
		err := body.Close()
		if err != nil {
			s.errCh <- fmt.Errorf("failed to close SSE stream: %w", err)
		}
	}(s.body)

	// Initialize OpenTelemetry span
	span := trace.SpanFromContext(context.Background())
	defer span.End()
	span.AddEvent("SSE stream started")

	if s.body == nil {
		s.errCh <- fmt.Errorf("SSE stream body is nil")
		return
	}

	var event SSEEvent

	for s.scanner.Scan() {
		line := bytes.TrimSpace(s.scanner.Bytes())
		if len(line) == 0 {
			// Empty line indicates the end of an event
			if len(event.Data) > 0 || event.Event != "" || event.ID != "" {
				s.eventCh <- event
				event = SSEEvent{} // reset event
			}
			continue
		}

		// Parse the line based on SSE format
		if bytes.HasPrefix(line, []byte(constant.SSEData+":")) {
			event.Data = append(event.Data, bytes.TrimSpace(line[5:])...)
			event.Data = append(event.Data, '\n')
		} else if bytes.HasPrefix(line, []byte(constant.SSEEvent+":")) {
			event.Event = string(bytes.TrimSpace(line[6:]))
		} else if bytes.HasPrefix(line, []byte(constant.SSEId+":")) {
			event.ID = string(bytes.TrimSpace(line[3:]))
		}

		span.AddEvent("SSE event received", trace.WithAttributes(
			attribute.String("event.type", event.Event),
			attribute.Int("data.length", len(event.Data)),
		))
	}

	// Check for errors after scanning
	if err := s.scanner.Err(); err != nil {
		s.errCh <- fmt.Errorf("SSE stream read error: %w", err)
		span.RecordError(err)
	} else {
		s.errCh <- io.EOF // indicate end of stream
	}
}

// Interface methods for SSEReader
func (s *SSEReader) Events() <-chan SSEEvent { return s.eventCh }
func (s *SSEReader) Err() <-chan error       { return s.errCh }
func (s *SSEReader) Close() error            { return s.body.Close() }
