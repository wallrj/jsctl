package components

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1core "k8s.io/api/core/v1"
)

func TestCertManagerApproverPolicy(t *testing.T) {
	var err error
	data, err := os.ReadFile("fixtures/cert-manager-approver-policy.json")
	require.NoError(t, err)

	var pod v1core.Pod

	err = json.Unmarshal(data, &pod)
	require.NoError(t, err)

	var status CertManagerApproverPolicyStatus

	md := &MatchData{
		Pods: []v1core.Pod{pod},
	}

	found, err := status.Match(md)
	require.NoError(t, err)
	require.True(t, found)

	assert.Equal(t, "cert-manager-approver-policy", status.Name())
	assert.Equal(t, "jetstack-secure", status.Namespace())
	assert.Equal(t, "v0.4.0", status.Version())
}
