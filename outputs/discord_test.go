package outputs

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"testing"

	"github.com/kubearmor/sidekick/types"
)

func TestNewDiscordPayload(t *testing.T) {
	// 1. Setup
	inputPayload := types.KubearmorPayload{}
	err := json.Unmarshal([]byte(TestInput), &inputPayload)
	if err != nil {
		t.Fatalf("Failed to unmarshal test input: %v", err)
	}

	config := &types.Configuration{
		Discord: types.DiscordOutputConfig{},
	}

	// 2. Execution
	result := newDiscordPayload(inputPayload, config)

	// 3. Assertion
	expectedAvatarURL := DefaultIconURL

	if result.AvatarURL != expectedAvatarURL {
		t.Errorf("Expected AvatarURL %v but got %v", expectedAvatarURL, result.AvatarURL)
	}

	if len(result.Embeds) != 1 {
		t.Fatalf("Expected exactly 1 embed but got %d", len(result.Embeds))
	}

	expectedEmbedDescription := inputPayload.EventType
	if result.Embeds[0].Description != expectedEmbedDescription {
		t.Errorf("Expected embed description %v but got %v", expectedEmbedDescription, result.Embeds[0].Description)
	}

	expectedEmbedFields := make([]discordEmbedFieldPayload, 0)
	for key, val := range inputPayload.OutputFields {
		var fieldValue string
		switch v := val.(type) {
		case string:
			if v == "" {
				continue
			}
			fieldValue = fmt.Sprintf("```%s```", v)
		default:
			fieldValue = fmt.Sprintf("```%v```", v)
		}
		expectedEmbedFields = append(expectedEmbedFields, discordEmbedFieldPayload{key, fieldValue, true})
	}

	expectedEmbedFields = append(expectedEmbedFields, discordEmbedFieldPayload{Hostname, inputPayload.Hostname, true})
	expectedEmbedFields = append(expectedEmbedFields, discordEmbedFieldPayload{Time, fmt.Sprint(inputPayload.Timestamp), true})

	sortFields(expectedEmbedFields)
	sortFields(result.Embeds[0].Fields)

	if !reflect.DeepEqual(result.Embeds[0].Fields, expectedEmbedFields) {
		t.Errorf("Expected embed fields %+v but got %+v", expectedEmbedFields, result.Embeds[0].Fields)
	}
}

func sortFields(fields []discordEmbedFieldPayload) {
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].Name < fields[j].Name
	})
}

// Content:   "",
// AvatarURL: "https://github.com/kubearmor/KubeArmor/assets/47106543/2db0b636-5c82-49c0-bf7d-535e4ad0a991",
// Embeds: []discordEmbedPayload{
// 	{
// 		Title:       "",
// 		Description: "MatchedPolicy",
// 		Color:       "", // Assuming the color is based on the EventType or Severity, this needs adjustments
// 		Fields: []discordEmbedFieldPayload{
// 			{Name: "ATags", Value: "ATag1,ATag2", Inline: true},
// 			{Name: "ClusterName", Value: "default", Inline: true},
// 			{Name: "PodName", Value: "wordpress-7c966b5d85-xvsrl", Inline: true},
// 			{Name: "Labels", Value: "app=wordpress", Inline: true},
// 			{Name: "HostPPID", Value: "102114", Inline: true},
// 			{Name: "ProcessName", Value: "/bin/ls", Inline: true},
// 			{Name: "UID", Value: "1001", Inline: true},
// 			{Name: "OwnerRef", Value: "Deployment", Inline: true},
// 			{Name: "OwnerNamespace", Value: "wordpress-mysql", Inline: true},
// 			{Name: "Timestamp", Value: "1631542902", Inline: true},
// 			{Name: "NamespaceName", Value: "wordpress-mysql", Inline: true},
// 			{Name: "ContainerID", Value: "80eead8fb840e9f3f3b1bea94bb202a798b92ad8ba4e0c92f52c4027dab98e73", Inline: true},
// 			{Name: "Hostname", Value: "gke-kubearmor-prerelease-default-pool-6ad71e07-cd8r", Inline: true},
// 			{Name: "PID", Value: "217", Inline: true},
// 			{Name: "ParentProcessName", Value: "/bin/bash", Inline: true},
// 			{Name: "Severity", Value: "Medium", Inline: true},
// 			{Name: "OwnerName", Value: "wordpress", Inline: true},
// 			{Name: "UpdatedTime", Value: "2023-09-13T15:35:02Z", Inline: true},
// 			{Name: "ContainerName", Value: "wordpress", Inline: true},
// 			{Name: "PPID", Value: "203", Inline: true},
// 			{Name: "Resource", Value: "/var", Inline: true},
// 			{Name: "Enforcer", Value: "eBPF Monitor", Inline: true},
// 			{Name: "ContainerImage", Value: "docker.io/library/wordpress:4.8-apache@sha256:6216f64ab88fc51d311e38c7f69ca3f9aaba621492b4f1fa93ddf63093768845", Inline: true},
// 			{Name: "Data", Value: "syscall=SYS_OPENAT fd=-100 flags=O_RDONLY|O_NONBLOCK|O_DIRECTORY|O_CLOEXEC", Inline: true},
// 			{Name: "Tags", Value: "Tag1,Tag2", Inline: true},
// 			{Name: "Message", Value: "Policy Matched", Inline: true},
// 			{Name: "Source", Value: "/bin/ls", Inline: true},
// 			{Name: "Result", Value: "Passed", Inline: true},
// 			{Name: "hostname", Value: "gke-kubearmor-prerelease-default-pool-6ad71e07-cd8r", Inline: true},
// 			{Name: "time", Value: "1631542902", Inline: true}},
