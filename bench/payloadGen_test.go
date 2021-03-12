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
