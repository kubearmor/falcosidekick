package outputs

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/falcosecurity/falcosidekick/types"
)

func TestNewTelegramPayload(t *testing.T) {
	expectedOutput := telegramPayload{
		Text:                  "*\\[Kubearmor\\] \\[Debug\\] Test rule*\n\n• *Time*: 2001\\-01\\-01 01:10:00 \\+0000 UTC\n• *Source*: syscalls\n• *Hostname*: test\\-host\n• *Tags*: test example \n• *Fields*:\n\t  • *proc\\.name*: kubearmor\n\t  • *proc\\.tty*: 1234\n\n\n**Output**: This is a test from kubearmor\n",
		ParseMode:             "MarkdownV2",
		DisableWebPagePreview: true,
		ChatID:                "-987654321",
	}

	var f types.KubearmorPayload
	require.Nil(t, json.Unmarshal([]byte(falcoTestInput), &f))
	config := &types.Configuration{
		Telegram: types.TelegramConfig{
			ChatID: "-987654321",
		},
	}

	output := newTelegramPayload(f, config)
	require.Equal(t, expectedOutput, output)
}
