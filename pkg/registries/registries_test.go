package registries

import (
	"encoding/json"
	"testing"

	config "github.com/openshift/api/config/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	core "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/credentialprovider"
)

func TestGetMergedMirrorSets(t *testing.T) {
	testCases := []struct {
		name           string
		inputIDMS      []config.ImageDigestMirrorSet
		inputITMS      []config.ImageTagMirrorSet
		expectedOutput []mirrorSet
	}{
		{
			name:           "No items",
			inputIDMS:      []config.ImageDigestMirrorSet{},
			inputITMS:      []config.ImageTagMirrorSet{},
			expectedOutput: []mirrorSet{},
		},
		{
			name: "One IDMS item",
			inputIDMS: []config.ImageDigestMirrorSet{
				{
					Spec: config.ImageDigestMirrorSetSpec{
						ImageDigestMirrors: []config.ImageDigestMirrors{
							{
								Source:             "source1",
								Mirrors:            []config.ImageMirror{"mirror1"},
								MirrorSourcePolicy: config.AllowContactingSource,
							},
						},
					},
				},
			},
			inputITMS: []config.ImageTagMirrorSet{},
			expectedOutput: []mirrorSet{
				{
					source:             "source1",
					mirrors:            []mirror{{host: "mirror1", resolveTags: false}},
					mirrorSourcePolicy: config.AllowContactingSource,
				},
			},
		},
		{
			name:      "One ITMS item",
			inputIDMS: []config.ImageDigestMirrorSet{},
			inputITMS: []config.ImageTagMirrorSet{
				{
					Spec: config.ImageTagMirrorSetSpec{
						ImageTagMirrors: []config.ImageTagMirrors{
							{
								Source:             "source2",
								Mirrors:            []config.ImageMirror{"mirror2"},
								MirrorSourcePolicy: config.AllowContactingSource,
							},
						},
					},
				},
			},
			expectedOutput: []mirrorSet{
				{
					source:             "source2",
					mirrors:            []mirror{{host: "mirror2", resolveTags: true}},
					mirrorSourcePolicy: config.AllowContactingSource,
				},
			},
		},
		{
			name: "1 IDMS and 1 ITMS",
			inputIDMS: []config.ImageDigestMirrorSet{
				{
					Spec: config.ImageDigestMirrorSetSpec{
						ImageDigestMirrors: []config.ImageDigestMirrors{
							{
								Source:             "https://source1.local:5000",
								Mirrors:            []config.ImageMirror{"mirror1"},
								MirrorSourcePolicy: config.AllowContactingSource,
							},
						},
					},
				},
			},
			inputITMS: []config.ImageTagMirrorSet{
				{
					Spec: config.ImageTagMirrorSetSpec{
						ImageTagMirrors: []config.ImageTagMirrors{
							{
								Source:             "source2",
								Mirrors:            []config.ImageMirror{"mirror2"},
								MirrorSourcePolicy: config.AllowContactingSource,
							},
						},
					},
				},
			},
			expectedOutput: []mirrorSet{
				{
					source:             "source1.local:5000",
					mirrors:            []mirror{{host: "mirror1", resolveTags: false}},
					mirrorSourcePolicy: config.AllowContactingSource,
				},
				{
					source:             "source2",
					mirrors:            []mirror{{host: "mirror2", resolveTags: true}},
					mirrorSourcePolicy: config.AllowContactingSource,
				},
			},
		},
		{
			name: "Mix of overlapping IDMS and 1 ITMS",
			inputIDMS: []config.ImageDigestMirrorSet{
				{
					Spec: config.ImageDigestMirrorSetSpec{
						ImageDigestMirrors: []config.ImageDigestMirrors{
							{
								Source:             "vmc.ci.openshift.org/ci-op/pipeline",
								Mirrors:            []config.ImageMirror{"devcluster.openshift.com:5000/pipeline"},
								MirrorSourcePolicy: config.AllowContactingSource,
							},
						},
					},
				},
			},
			inputITMS: []config.ImageTagMirrorSet{
				{
					Spec: config.ImageTagMirrorSetSpec{
						ImageTagMirrors: []config.ImageTagMirrors{
							{
								Source:             "docker://mcr.microsoft.com/oss/kubernetes/pause",
								Mirrors:            []config.ImageMirror{"quay.io/testuser/oss/kubernetes/pause"},
								MirrorSourcePolicy: config.AllowContactingSource,
							},
							{
								Source:             "mcr.microsoft.com/powershell",
								Mirrors:            []config.ImageMirror{"quay.io/testuser/testnamespace/powershell"},
								MirrorSourcePolicy: config.AllowContactingSource,
							},
							{
								Source:             "registry.k8s.io/sig-storage/csi-provisioner",
								Mirrors:            []config.ImageMirror{"devcluster.openshift.com:5000/sig-storage/csi-provisioner"},
								MirrorSourcePolicy: config.AllowContactingSource,
							},
							{
								Source:             "registry.access.redhat.com/ubi9/ubi-minimal",
								Mirrors:            []config.ImageMirror{"devcluster.openshift.com:5000/ubi9/ubi-minimal"},
								MirrorSourcePolicy: config.AllowContactingSource,
							},
							{
								Source:             "registry.access.redhat.com/ubi8/ubi-minimal",
								Mirrors:            []config.ImageMirror{"random.io/ubi8/ubi-minimal"},
								MirrorSourcePolicy: config.NeverContactSource,
							},
						},
					},
				},
			},
			expectedOutput: []mirrorSet{
				{
					source: "mcr.microsoft.com",
					mirrors: []mirror{
						{
							host:        "quay.io/testuser",
							resolveTags: true,
						},
						{
							host:        "quay.io/testuser/testnamespace",
							resolveTags: true,
						},
					},
					mirrorSourcePolicy: "AllowContactingSource"},
				{
					source: "registry.access.redhat.com",
					mirrors: []mirror{
						{
							host:        "devcluster.openshift.com:5000",
							resolveTags: true,
						},
						{
							host:        "random.io",
							resolveTags: true,
						},
					},
					mirrorSourcePolicy: "NeverContactSource",
				},
				{
					source: "registry.k8s.io",
					mirrors: []mirror{
						{
							host:        "devcluster.openshift.com:5000",
							resolveTags: true,
						},
					},
					mirrorSourcePolicy: "AllowContactingSource",
				},
				{
					source: "vmc.ci.openshift.org",
					mirrors: []mirror{
						{
							host:        "devcluster.openshift.com:5000",
							resolveTags: false,
						},
					},
					mirrorSourcePolicy: "AllowContactingSource",
				},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			out := getMergedMirrorSets(test.inputIDMS, test.inputITMS)
			assert.Equal(t, test.expectedOutput, out)
			//assert.True(t, reflect.DeepEqual(out, test.expectedOutput))
		})
	}
}

func TestGenerateConfig(t *testing.T) {
	pullSecret := core.Secret{
		Data: map[string][]byte{
			core.DockerConfigJsonKey: []byte(`{"auths":{"mirror.example.com":{"auth":"dXNlcjpwYXNz"},"mirror.example.net":{"auth":"dXNlcm5hbWU6cGFzc3dvcmQ="}}}`),
		},
	}

	testCases := []struct {
		name           string
		input          mirrorSet
		expectedOutput string
	}{
		{
			name: "empty mirrors",
			input: mirrorSet{
				source:             "registry.access.redhat.com/ubi9",
				mirrors:            []mirror{},
				mirrorSourcePolicy: config.AllowContactingSource,
			},
			expectedOutput: "",
		},
		{
			name: "basic one digest mirror",
			input: mirrorSet{
				source: "registry.access.redhat.com",
				mirrors: []mirror{
					{host: "example.io/example", resolveTags: false},
				},
				mirrorSourcePolicy: config.AllowContactingSource,
			},
			expectedOutput: `server = "https://registry.access.redhat.com/v2"

override_path = true

[host."https://example.io/v2/example"]
  capabilities = ["pull"]
  override_path = true
`,
		},
		{
			name: "basic one tag mirror",
			input: mirrorSet{
				source: "registry.access.redhat.com",
				mirrors: []mirror{
					{host: "example.io/example", resolveTags: true},
				},
				mirrorSourcePolicy: config.AllowContactingSource,
			},
			expectedOutput: `server = "https://registry.access.redhat.com/v2"

override_path = true

[host."https://example.io/v2/example"]
  capabilities = ["pull", "resolve"]
  override_path = true
`,
		},
		{
			name: "one digest mirror never contact source",
			input: mirrorSet{
				source: "registry.access.redhat.com",
				mirrors: []mirror{
					{host: "example.io/example", resolveTags: false},
				},
				mirrorSourcePolicy: config.NeverContactSource,
			},
			expectedOutput: `server = "https://example.io/v2/example"

override_path = true

[host."https://example.io/v2/example"]
  capabilities = ["pull"]
  override_path = true
`,
		},
		{
			name: "tags mirror never contact source",
			input: mirrorSet{
				source: "registry.access.redhat.com",
				mirrors: []mirror{
					{host: "example.io/example", resolveTags: true},
				},
				mirrorSourcePolicy: config.NeverContactSource,
			},
			expectedOutput: `server = "https://example.io/v2/example"

override_path = true

[host."https://example.io/v2/example"]
  capabilities = ["pull", "resolve"]
  override_path = true
`,
		},
		{
			name: "multiple mirrors",
			input: mirrorSet{
				source: "registry.access.redhat.com",
				mirrors: []mirror{
					{host: "example.io/example", resolveTags: false},
					{host: "mirror.example.com/redhat", resolveTags: false},
					{host: "mirror.example.net", resolveTags: true},
				},
				mirrorSourcePolicy: config.AllowContactingSource,
			},
			expectedOutput: `server = "https://registry.access.redhat.com/v2"

override_path = true

[host."https://example.io/v2/example"]
  capabilities = ["pull"]
  override_path = true

[host."https://mirror.example.com/v2/redhat"]
  capabilities = ["pull"]
  override_path = true
  [host."https://mirror.example.com/v2/redhat".header]
    authorization = "Basic dXNlcjpwYXNz"

[host."https://mirror.example.net/v2"]
  capabilities = ["pull", "resolve"]
  override_path = true
  [host."https://mirror.example.net/v2".header]
    authorization = "Basic dXNlcm5hbWU6cGFzc3dvcmQ="
`,
		},
		{
			name: "multiple mirrors never contact source",
			input: mirrorSet{
				source: "registry.access.redhat.com",
				mirrors: []mirror{
					{host: "example.io/example", resolveTags: false},
					{host: "mirror.example.com/redhat", resolveTags: false},
					{host: "mirror.example.net", resolveTags: true},
				},
				mirrorSourcePolicy: config.NeverContactSource,
			},
			expectedOutput: `server = "https://example.io/v2/example"

override_path = true

[host."https://example.io/v2/example"]
  capabilities = ["pull"]
  override_path = true

[host."https://mirror.example.com/v2/redhat"]
  capabilities = ["pull"]
  override_path = true
  [host."https://mirror.example.com/v2/redhat".header]
    authorization = "Basic dXNlcjpwYXNz"

[host."https://mirror.example.net/v2"]
  capabilities = ["pull", "resolve"]
  override_path = true
  [host."https://mirror.example.net/v2".header]
    authorization = "Basic dXNlcm5hbWU6cGFzc3dvcmQ="
`,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			var secretsConfig credentialprovider.DockerConfigJSON
			err := json.Unmarshal(pullSecret.Data[core.DockerConfigJsonKey], &secretsConfig)
			require.NoError(t, err)

			out := test.input.generateConfig(secretsConfig)
			assert.Equal(t, test.expectedOutput, out)
		})
	}
}

func TestMergeMirrorSets(t *testing.T) {
	testCases := []struct {
		name  string
		input []mirrorSet
		// expectedOutput's sources and mirror orders matter since result is expected to be sorted alphabetically
		expectedOutput []mirrorSet
	}{
		{
			name:           "empty mirrorset",
			input:          []mirrorSet{},
			expectedOutput: []mirrorSet{},
		},
		{
			name: "same source but different mirrors",
			input: []mirrorSet{
				{
					source: "registry.access.redhat.com/ubi9/ubi-minimal",
					mirrors: []mirror{
						{"example.io/example/ubi-minimal", false},
						{"example.com/example/ubi-minimal", true},
					},
				},
				{
					source: "registry.access.redhat.com/ubi9/ubi-minimal",
					mirrors: []mirror{
						{"mirror.example.net/image", false},
						{"mirror.example.com/redhat", true},
					},
				},
			},
			expectedOutput: []mirrorSet{
				{
					source: "registry.access.redhat.com/ubi9/ubi-minimal",
					mirrors: []mirror{
						{"example.com/example/ubi-minimal", true},
						{"example.io/example/ubi-minimal", false},
						{"mirror.example.com/redhat", true},
						{"mirror.example.net/image", false},
					},
				},
			},
		},
		{
			name: "same source, ensuring mirrorSourcePolicy is handled correctly",
			input: []mirrorSet{
				{
					source:             "registry.access.redhat.com/ubi9/ubi-minimal",
					mirrorSourcePolicy: config.NeverContactSource,
				},
				{
					source:             "registry.access.redhat.com/ubi9/ubi-minimal",
					mirrorSourcePolicy: config.AllowContactingSource,
				},
				{
					source:             "quay.io/openshift-release-dev/ocp-release",
					mirrorSourcePolicy: config.AllowContactingSource,
				},
				{
					source:             "quay.io/openshift-release-dev/ocp-release",
					mirrorSourcePolicy: config.AllowContactingSource,
				},
			},
			expectedOutput: []mirrorSet{
				{
					source:             "quay.io/openshift-release-dev/ocp-release",
					mirrorSourcePolicy: config.AllowContactingSource,
				},
				{
					source:             "registry.access.redhat.com/ubi9/ubi-minimal",
					mirrorSourcePolicy: config.NeverContactSource,
				},
			},
		},
		{
			name: "same source and duplicated mirrors, ensuring resolveTags is handled correctly",
			input: []mirrorSet{
				{
					source: "registry.access.redhat.com/ubi9/ubi-minimal",
					mirrors: []mirror{
						{"mirror.example.net/image", false},
						{"mirror.example.com/redhat", false},
					},
				},
				{
					source: "registry.access.redhat.com/ubi9/ubi-minimal",
					mirrors: []mirror{
						{"mirror.example.net/image", false},
						{"mirror.example.com/redhat", true},
					},
				},
			},
			expectedOutput: []mirrorSet{
				{
					source: "registry.access.redhat.com/ubi9/ubi-minimal",
					mirrors: []mirror{
						{"mirror.example.com/redhat", true},
						{"mirror.example.net/image", false},
					},
				},
			},
		},
		{
			name: "different sources",
			input: []mirrorSet{
				{
					source: "registry.access.redhat.com/ubi9/ubi-minimal",
					mirrors: []mirror{
						{"mirror.example.com/redhat", false},
						{"mirror.example.net/image", false},
					},
				},
				{
					source: "quay.io/openshift-release-dev/ocp-release",
					mirrors: []mirror{
						{"mirror.registry.com:443/ocp/release", false},
					},
				},
			},
			expectedOutput: []mirrorSet{
				{
					source: "quay.io/openshift-release-dev/ocp-release",
					mirrors: []mirror{
						{"mirror.registry.com:443/ocp/release", false},
					},
				},
				{
					source: "registry.access.redhat.com/ubi9/ubi-minimal",
					mirrors: []mirror{
						{"mirror.example.com/redhat", false},
						{"mirror.example.net/image", false},
					},
				},
			},
		},
	}
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			out := mergeMirrorSets(test.input)
			assert.Equal(t, out, test.expectedOutput)
		})
	}
}

func TestMergeMirrors(t *testing.T) {
	testCases := []struct {
		name            string
		mirrorsA        []mirror
		mirrorsB        []mirror
		expectedMirrors []mirror
	}{
		{
			name:            "no mirrors",
			mirrorsA:        []mirror{},
			mirrorsB:        []mirror{},
			expectedMirrors: []mirror{},
		},
		{
			name: "one empty slice",
			mirrorsA: []mirror{
				{host: "openshift.com", resolveTags: false},
			},
			mirrorsB: []mirror{},
			expectedMirrors: []mirror{
				{host: "openshift.com", resolveTags: false},
			},
		},
		{
			name: "duplicate mirror",
			mirrorsA: []mirror{
				{host: "openshift.com", resolveTags: false},
			},
			mirrorsB: []mirror{
				{host: "openshift.com", resolveTags: false},
			},
			expectedMirrors: []mirror{
				{host: "openshift.com", resolveTags: false},
			},
		},
		{
			name: "duplicate host but different resolveTags",
			mirrorsA: []mirror{
				{host: "openshift.com", resolveTags: false},
			},
			mirrorsB: []mirror{
				{host: "openshift.com", resolveTags: true},
			},
			expectedMirrors: []mirror{
				{host: "openshift.com", resolveTags: true},
			},
		},
		{
			name: "different mirrors",
			mirrorsA: []mirror{
				{host: "redhat.com", resolveTags: false},
			},
			mirrorsB: []mirror{
				{host: "openshift.com", resolveTags: true},
			},
			expectedMirrors: []mirror{
				{host: "redhat.com", resolveTags: false},
				{host: "openshift.com", resolveTags: true},
			},
		},
		{
			name: "multiple mirrors",
			mirrorsA: []mirror{
				{host: "redhat.com", resolveTags: false},
				{host: "openshift.com", resolveTags: true},
				{host: "example.test.io", resolveTags: true},
			},
			mirrorsB: []mirror{
				{host: "openshift.com", resolveTags: true},
				{host: "example.test.io", resolveTags: true},
			},
			expectedMirrors: []mirror{
				{host: "redhat.com", resolveTags: false},
				{host: "openshift.com", resolveTags: true},
				{host: "example.test.io", resolveTags: true},
			},
		},
	}
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			out := mergeMirrors(test.mirrorsA, test.mirrorsB)
			assert.Equal(t, len(out), len(test.expectedMirrors))
			for _, m := range test.expectedMirrors {
				assert.Contains(t, out, m)
			}
		})
	}
}

func TestExtractMirrorURL(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		mirror   string
		expected string
	}{
		{
			name:     "Exact match",
			source:   "example.com/path/to/resource",
			mirror:   "example.com/path/to/resource",
			expected: "",
		},
		{
			name:     "Different domain",
			source:   "example.com/path/to/resource",
			mirror:   "example.org/path/to/resource",
			expected: "example.org",
		},
		{
			name:     "Last letter equal but no namespace match",
			source:   "example.com/path/to/resource/x",
			mirror:   "example.com/path/to/resourceax",
			expected: "example.com/path/to/resourceax",
		},
		{
			name:     "Different tags",
			source:   "mcr.microsoft.com/powershell:lts-nanoserver-ltsc2022",
			mirror:   "quay.io/powershell:23",
			expected: "quay.io/powershell:23",
		},
		{
			name:     "Different domain with tag",
			source:   "mcr.microsoft.com/powershell:lts-nanoserver-ltsc2022",
			mirror:   "quay.io/powershell:lts-nanoserver-ltsc2022",
			expected: "quay.io",
		},
		{
			name:     "1 leading namespace",
			source:   "mcr.microsoft.com/powershell:lts-nanoserver-ltsc2022",
			mirror:   "quay.io/random_namespace/powershell:lts-nanoserver-ltsc2022",
			expected: "quay.io/random_namespace",
		},
		{
			name:     "2 leading namespaces",
			source:   "mcr.microsoft.com/windows/servercore:ltsc2022",
			mirror:   "quay.io/mohashai/random_namespace/windows/servercore:ltsc2022",
			expected: "quay.io/mohashai/random_namespace",
		},
		{
			name:     "Matching higher level namespace",
			source:   "foo1/fah/different/foo2",
			mirror:   "foo1/fah/something/foo2",
			expected: "foo1/fah/something",
		},
		{
			name:     "Matching higher level namespace with longer source",
			source:   "foo1/fah/hello/different/foo2",
			mirror:   "foo1/fah/something/foo2",
			expected: "foo1/fah/something",
		},
		{
			name:     "Matching higher level namespace with longer mirror",
			source:   "foo1/fah/different/foo2",
			mirror:   "foo1/fah/hello/something/foo2",
			expected: "foo1/fah/hello/something",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractMirrorURL(tt.source, tt.mirror)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestExtractOrgPath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "registry.local/org",
			expected: "org",
		},
		{
			input:    "registry.local/org/sub_org",
			expected: "org/sub_org",
		},
		{
			input:    "registry.local",
			expected: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := extractRegistryOrgPath(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
