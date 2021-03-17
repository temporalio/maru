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
	"math/rand"
	"regexp"
	"strconv"
)

func buildPayload(params interface{}) interface{} {
	olds, ok := params.(map[string]interface{})
	if !ok {
		return params
	}
	news := map[string]interface{}{}
	for k, v := range olds {
		if str, ok := v.(string); ok {
			if newValue, ok := eval(str); ok {
				news[k] = newValue
				continue
			}
		}
		news[k] = v
	}
	return news
}

var randomRegex = regexp.MustCompile(`\$RANDOM\(([0-9]+)\)`)
var randomNormRegex = regexp.MustCompile(`\$RANDOM_NORM\(([0-9]+),([0-9]+)\)`)

func eval(payload string) (string, bool) {
	length := 0

	match := randomNormRegex.FindStringSubmatch(payload)
	if len(match) > 2 {
		if mu, err := strconv.Atoi(match[1]); err == nil {
			if sigma, err := strconv.Atoi(match[2]); err == nil {
				length = normalInverse(mu, sigma)
			}
		}
	}

	match = randomRegex.FindStringSubmatch(payload)
	if len(match) > 1 {
		if v, err := strconv.Atoi(match[1]); err == nil {
			length = v
		}
	}

	if length > 0 {
		return generateRandomPayload(length), true
	}

	return payload, false
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func generateRandomPayload(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func normalInverse(mu int, sigma int) int {
	return int(rand.NormFloat64()*float64(sigma) + float64(mu))
}
