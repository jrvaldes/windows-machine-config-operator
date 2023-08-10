package ignition

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseKubeletArgs(t *testing.T) {
	unitContents := `[Unit]
Description=Kubernetes Kubelet
Wants=rpc-statd.service network-online.target
Requires=crio.service kubelet-auto-node-size.service
After=network-online.target crio.service kubelet-auto-node-size.service
After=ostree-finalize-staged.service

[Service]
Type=notify
ExecStartPre=/bin/mkdir --parents /etc/kubernetes/manifests
ExecStartPre=/bin/rm -f /var/lib/kubelet/cpu_manager_state
ExecStartPre=/bin/rm -f /var/lib/kubelet/memory_manager_state
EnvironmentFile=/etc/os-release
EnvironmentFile=-/etc/kubernetes/kubelet-workaround
EnvironmentFile=-/etc/kubernetes/kubelet-env
EnvironmentFile=/etc/node-sizing.env

ExecStart=/usr/local/bin/kubenswrapper \
    /usr/bin/kubelet \
      --config=/etc/kubernetes/kubelet.conf \
      --bootstrap-kubeconfig=/etc/kubernetes/kubeconfig \
      --kubeconfig=/var/lib/kubelet/kubeconfig \
      --container-runtime=remote \
      --container-runtime-endpoint=/var/run/crio/crio.sock \
      --runtime-cgroups=/system.slice/crio.service \
      --node-labels=node-role.kubernetes.io/worker,node.openshift.io/os_id=${ID} \
      --node-ip=${KUBELET_NODE_IP} \
      --minimum-container-ttl-duration=6m0s \
      --volume-plugin-dir=/etc/kubernetes/kubelet-plugins/volume/exec \
      --cloud-provider=azure \
      --cloud-config=/etc/kubernetes/cloud.conf \
      --hostname-override=${KUBELET_NODE_NAME} \
      --provider-id=${KUBELET_PROVIDERID} \
      --pod-infra-container-image=quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:204ff33466fefe3068e49d6b46583e164fcb2f419f5e55af5f58539fdf55d931 \
      --system-reserved=cpu=${SYSTEM_RESERVED_CPU},memory=${SYSTEM_RESERVED_MEMORY} \
      --v=${KUBELET_LOG_LEVEL}

Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
`
	args, err := parseKubeletArgs(unitContents)
	require.NoError(t, err)
	require.Contains(t, args, CloudProviderOption)
	assert.Equal(t, "azure", args[CloudProviderOption])
	require.Contains(t, args, CloudConfigOption)
	assert.Equal(t, "/etc/kubernetes/cloud.conf", args[CloudConfigOption])

}

func TestGetEscapedData(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want string
	}{
		{
			name: "empty data",
			data: []byte(""),
			want: "data:,",
		},
		{
			name: "with space",
			data: []byte("a b"),
			want: "data:,a%20b",
		},
		{
			name: "with new line",
			data: []byte("a\nb"),
			want: "data:,a%0Ab",
		},
		{
			name: "with tabulation",
			data: []byte("a\tb"),
			want: "data:,a%09b",
		},
		{
			name: "CA like",
			data: []byte("-----BEGIN CERTIFICATE-----\nMIIDMDCCAhigAwIBAgIIdhr1kYMDqpYwDQYJKoZIhvcNAQELBQAwNjESMBAGA1UE\nREDACTED\nJ09L8w==\n-----END CERTIFICATE-----\n"),
			want: "data:,-----BEGIN%20CERTIFICATE-----%0AMIIDMDCCAhigAwIBAgIIdhr1kYMDqpYwDQYJKoZIhvcNAQELBQAwNjESMBAGA1UE%0AREDACTED%0AJ09L8w%3D%3D%0A-----END%20CERTIFICATE-----%0A",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, getEscapedData(tt.data))
		})
	}
}
