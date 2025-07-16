// Copyright (c) 2020-2025 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package atomic

import (
	"encoding/base64"
	"encoding/json"
)

// MarshalJSON encodes the wrapped []byte as a base64 string.
//
// This makes it encodable as JSON.
func (b *Bytes) MarshalJSON() ([]byte, error) {
	data := b.Load()
	if data == nil {
		return []byte("null"), nil
	}
	encoded := base64.StdEncoding.EncodeToString(data)
	return json.Marshal(encoded)
}

// UnmarshalJSON decodes a base64 string and replaces the wrapped []byte with it.
//
// This makes it decodable from JSON.
func (b *Bytes) UnmarshalJSON(text []byte) error {
	var encoded string
	if err := json.Unmarshal(text, &encoded); err != nil {
		return err
	}

	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return err
	}

	b.Store(decoded)
	return nil
}
