// The MIT License
//
// Copyright (c) 2021 Temporal Technologies Inc.  All rights reserved.
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

package bench

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBuildParametersNoChangeInUnrelatedStruct(t *testing.T) {
	x := map[string]interface{}{"Hello": 123}
	actual := buildPayload(x)
	assert.Equal(t, x, actual)
}

func TestBuildParametersNoChangeInFixedPayload(t *testing.T) {
	x := map[string]interface{}{"payload": "123"}
	actual := buildPayload(x)
	assert.Equal(t, x, actual)
}

func TestBuildParametersRandomPayload(t *testing.T) {
	x := map[string]interface{}{"payload": "$RANDOM(10)"}
	y := buildPayload(x).(map[string]interface{})
	assert.NotEqual(t, x["payload"], y["payload"])
	assert.Equal(t, 10, len(y["payload"].(string)))
}

func TestBuildParametersRandomNormalPayload(t *testing.T) {
	x := map[string]interface{}{"payload": "$RANDOM_NORM(80,10)"}
	y := buildPayload(x).(map[string]interface{})
	assert.NotEqual(t, x["payload"], y["payload"])
	actual := len(y["payload"].(string))
	assert.Greater(t, actual, 0)
	assert.Less(t, actual, 160)
}
