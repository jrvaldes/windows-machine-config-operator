package windows

import (
	"strings"
	"testing"

	config "github.com/openshift/api/config/v1"
	"github.com/stretchr/testify/assert"

	"github.com/openshift/windows-machine-config-operator/pkg/nodeconfig/payload"
)

func TestGetFilesToTransfer(t *testing.T) {
	testCases := []struct {
		name     string
		platform *config.PlatformType
	}{
		{
			name:     "test AWS",
			platform: func() *config.PlatformType { t := config.AWSPlatformType; return &t }(),
		},
		{
			name:     "test Azure",
			platform: func() *config.PlatformType { t := config.AzurePlatformType; return &t }(),
		},
		{
			name:     "test Nil",
			platform: nil,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			files := getFilesToTransfer(test.platform)
			if test.platform != nil && *test.platform == config.AzurePlatformType {
				file := files[payload.AzureCloudNodeManagerPath]
				assert.Equal(t, K8sDir, file)
			} else {
				_, exists := files[payload.AzureCloudNodeManagerPath]
				assert.False(t, exists)
			}
		})
	}
}

func TestSplitPath(t *testing.T) {
	testCases := []struct {
		name                string
		inputWindowsPath    string
		expectedOutDir      string
		expectedOutFileName string
	}{
		{
			name:                "empty input",
			inputWindowsPath:    "",
			expectedOutDir:      "",
			expectedOutFileName: "",
		},
		{
			name:                "filename only",
			inputWindowsPath:    "test.yml",
			expectedOutDir:      "",
			expectedOutFileName: "test.yml",
		},
		{
			name:                "directory only",
			inputWindowsPath:    "C:\\var\\log\\",
			expectedOutDir:      "C:\\var\\log\\",
			expectedOutFileName: "",
		},
		{
			name:                "full Windows path",
			inputWindowsPath:    "C:\\var\\log\\service.txt",
			expectedOutDir:      "C:\\var\\log\\",
			expectedOutFileName: "service.txt",
		},
		{
			name:                "full linux path",
			inputWindowsPath:    "/home/user/README.md",
			expectedOutDir:      "",
			expectedOutFileName: "/home/user/README.md",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			dir, fileName := SplitPath(test.inputWindowsPath)
			assert.Equal(t, test.expectedOutDir, dir)
			assert.Equal(t, test.expectedOutFileName, fileName)
		})
	}
}

func TestLogRunnerCmd(t *testing.T) {
	tests := []struct {
		name                string
		commandPath         string
		logfilePath         string
		logFileSize         string
		logFileAge          string
		flushInterval       string
		expectedContains    []string
		expectedNotContains []string
	}{
		{
			name:          "basic command with no optional parameters",
			commandPath:   KubeletPath,
			logfilePath:   KubeletLog,
			logFileSize:   "",
			logFileAge:    "",
			flushInterval: "",
			expectedContains: []string{
				KubeLogRunnerPath,
				"-log-file=" + KubeletLog,
				KubeletPath,
			},
			expectedNotContains: []string{
				"-log-file-size=",
				"-log-file-age=",
				"-flush-interval=",
			},
		},
		{
			name:          "command with log file size set",
			commandPath:   KubeProxyPath,
			logfilePath:   KubeProxyLog,
			logFileSize:   "100Mi",
			logFileAge:    "",
			flushInterval: "",
			expectedContains: []string{
				KubeLogRunnerPath,
				"-log-file=" + KubeProxyLog,
				"-log-file-size=100Mi",
				KubeProxyPath,
			},
			expectedNotContains: []string{
				"-log-file-age=",
				"-flush-interval=",
			},
		},
		{
			name:          "command with log file age set",
			commandPath:   KubeletPath,
			logfilePath:   KubeletLog,
			logFileSize:   "",
			logFileAge:    "24h",
			flushInterval: "",
			expectedContains: []string{
				KubeLogRunnerPath,
				"-log-file=" + KubeletLog,
				"-log-file-age=24h",
				KubeletPath,
			},
			expectedNotContains: []string{
				"-log-file-size=",
				"-flush-interval=",
			},
		},
		{
			name:          "command with flush interval set",
			commandPath:   KubeletPath,
			logfilePath:   KubeletLog,
			logFileSize:   "",
			logFileAge:    "",
			flushInterval: "5s",
			expectedContains: []string{
				KubeLogRunnerPath,
				"-log-file=" + KubeletLog,
				"-flush-interval=5s",
				KubeletPath,
			},
			expectedNotContains: []string{
				"-log-file-size=",
				"-log-file-age=",
			},
		},
		{
			name:          "command with all optional parameters set",
			commandPath:   KubeletPath,
			logfilePath:   KubeletLog,
			logFileSize:   "50Mi",
			logFileAge:    "48h",
			flushInterval: "10s",
			expectedContains: []string{
				KubeLogRunnerPath,
				"-log-file=" + KubeletLog,
				"-log-file-size=50Mi",
				"-log-file-age=48h",
				"-flush-interval=10s",
				KubeletPath,
			},
			expectedNotContains: []string{},
		},
		{
			name:          "WICD command with log-file-size and log-file-age",
			commandPath:   wicdPath,
			logfilePath:   WicdLog,
			logFileSize:   "200Mi",
			logFileAge:    "72h",
			flushInterval: "",
			expectedContains: []string{
				KubeLogRunnerPath,
				"-log-file=" + WicdLog,
				"-log-file-size=200Mi",
				"-log-file-age=72h",
				wicdPath,
			},
			expectedNotContains: []string{
				"-flush-interval=",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := LogRunnerCmd(tc.commandPath, tc.logfilePath, tc.logFileSize, tc.logFileAge, tc.flushInterval)

			for _, expected := range tc.expectedContains {
				assert.Contains(t, result, expected,
					"Command should contain: %s\nActual command: %s", expected, result)
			}
			for _, notExpected := range tc.expectedNotContains {
				assert.NotContains(t, result, notExpected,
					"Command should not contain: %s\nActual command: %s", notExpected, result)
			}

			// Verify ordering: KubeLogRunnerPath must come first, commandPath must come last
			assert.True(t, len(result) > 0, "Result should not be empty")
			assert.Equal(t, 0, strings.Index(result, KubeLogRunnerPath),
				"KubeLogRunnerPath must be at the start of the command")
			expectedSuffix := " " + tc.commandPath
			assert.True(t, len(result) >= len(expectedSuffix) &&
				result[len(result)-len(expectedSuffix):] == expectedSuffix,
				"Command path must be at the end of the command string.\nActual: %s", result)
		})
	}
}

func TestRmK8sFilesCmd(t *testing.T) {
	cmd := rmK8sFilesCmd()
	// Verify all three WICD-related files are excluded from deletion
	assert.Contains(t, cmd, wicdPath,
		"rmK8sFilesCmd should exclude WICD binary")
	assert.Contains(t, cmd, WICDKubeconfigPath,
		"rmK8sFilesCmd should exclude WICD kubeconfig")
	assert.Contains(t, cmd, KubeLogRunnerPath,
		"rmK8sFilesCmd should exclude kube-log-runner (WICD's registered service binary)")
	// Verify basic command structure
	assert.Contains(t, cmd, "-Exclude")
	assert.Contains(t, cmd, "Remove-Item -Force -Recurse")
	assert.Contains(t, cmd, K8sDir)
}

func TestLogRotationEnvVars(t *testing.T) {
	tests := []struct {
		name                               string
		envSize, envAge, envFlush          string
		wantSize, wantAge, wantFlush string
	}{
		{
			name:      "all unset",
			envSize:   "", envAge: "", envFlush: "",
			wantSize:  "", wantAge: "", wantFlush: "",
		},
		{
			name:      "all set",
			envSize:   "100Mi", envAge: "24h", envFlush: "5s",
			wantSize:  "100Mi", wantAge: "24h", wantFlush: "5s",
		},
		{
			name:      "whitespace is trimmed",
			envSize:   "  50Mi  ", envAge: " 1h ", envFlush: " 10s ",
			wantSize:  "50Mi", wantAge: "1h", wantFlush: "10s",
		},
		{
			name:      "only size set",
			envSize:   "200Mi", envAge: "", envFlush: "",
			wantSize:  "200Mi", wantAge: "", wantFlush: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("SERVICES_LOG_FILE_SIZE", tc.envSize)
			t.Setenv("SERVICES_LOG_FILE_AGE", tc.envAge)
			t.Setenv("SERVICES_LOG_FLUSH_INTERVAL", tc.envFlush)
			size, age, flush := logRotationEnvVars()
			assert.Equal(t, tc.wantSize, size)
			assert.Equal(t, tc.wantAge, age)
			assert.Equal(t, tc.wantFlush, flush)
		})
	}
}
