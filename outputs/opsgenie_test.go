package outputs

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/falcosecurity/falcosidekick/types"
)

func TestNewOpsgeniePayload(t *testing.T) {
	expectedOutput := opsgeniePayload{
		Message:     "This is a test from falcosidekick",
		Entity:      "Kubearmor",
		Description: "Test rule",
		Details: map[string]string{
			"hostname":  "test-host",
			"priority":  "Debug",
			"tags":      "test, example",
			"proc_name": "kubearmor",
			"rule":      "Test rule",
			"source":    "syscalls",
		},
		Priority: "P5",
	}

	var f types.KubearmorPayload
	require.Nil(t, json.Unmarshal([]byte(falcoTestInput), &f))
	output := newOpsgeniePayload(f, &types.Configuration{})

	require.Equal(t, output, expectedOutput)
}
