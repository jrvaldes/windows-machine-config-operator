package services

import (
	"testing"

	config "github.com/openshift/api/config/v1"
	"github.com/stretchr/testify/assert"
)

func TestGetHostnameCmd(t *testing.T) {
	tests := []struct {
		name         string
		platformType config.PlatformType
		expected     string
	}{
		{
			name:         "any platform",
			platformType: "",
			expected:     "",
		},
		{
			name:         "AWS platform",
			platformType: config.AWSPlatformType,
			expected:     "Get-EC2InstanceMetadata -Category LocalHostname",
		},
		{
			name:         "GCP platform",
			platformType: config.GCPPlatformType,
			expected:     "C:\\Temp\\gcp-get-hostname.ps1",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := getHostnameCmd(test.platformType)
			assert.Equal(t, test.expected, actual)
		})
	}
}