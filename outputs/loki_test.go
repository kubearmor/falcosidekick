package outputs

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kubearmor/sidekick/types"
)

func TestNewLokiPayload(t *testing.T) {
	expectedOutput := lokiPayload{
		Streams: []lokiStream{
			{
				Stream: map[string]string{
					"hostname": "test-host",
					"tags":     "test,example",
					"rule":     "Test rule",
					"source":   "syscalls",
					"priority": "Debug",
				},
				Values: []lokiValue{[]string{"978311400000000000", "This is a test from Kubearmor"}},
			},
		},
	}

	var f types.KubearmorPayload
	require.Nil(t, json.Unmarshal([]byte(falcoTestInput), &f))
	output := newLokiPayload(f, &types.Configuration{})

	require.Equal(t, output, expectedOutput)
}
