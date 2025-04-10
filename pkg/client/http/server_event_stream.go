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
	"io"
	"net/http"
)

import (
	"github.com/apache/dubbo-go-pixiu/pkg/common/constant"
)

type SSEReader struct {
	body    io.ReadCloser
	Scanner *bufio.Scanner
}

// IsSSEStream check if the response is a SSE stream
func IsSSEStream(resp *http.Response) bool {
	contentType := resp.Header.Get(constant.HeaderKeyContextType)
	return contentType == constant.HeaderValueTextEventStream
}

func NewSSEReader(body io.ReadCloser) *SSEReader {
	s := &SSEReader{
		body:    body,
		Scanner: bufio.NewScanner(body),
	}
	return s
}

func (s *SSEReader) Read() ([]byte, error) {
	if !s.Scanner.Scan() {
		return []byte(""), io.EOF
	}
	line := s.Scanner.Bytes()
	return line, nil
}

func (s *SSEReader) Close() error {
	return s.body.Close()
}
