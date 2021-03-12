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
