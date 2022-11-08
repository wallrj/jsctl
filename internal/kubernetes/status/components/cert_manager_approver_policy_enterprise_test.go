package components

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
)

func TestCertManagerApproverPolicyEnterprise(t *testing.T) {
	var err error
	data, err := os.ReadFile("fixtures/cert-manager-approver-policy-enterprise.json")
	require.NoError(t, err)

	var pod v1.Pod

	err = json.Unmarshal(data, &pod)
	require.NoError(t, err)

	var status CertManagerApproverPolicyEnterpriseStatus

	found, err := status.Match(&pod)
	require.NoError(t, err)
	require.True(t, found)

	assert.Equal(t, "cert-manager-approver-policy-enterprise", status.Name())
	assert.Equal(t, "example", status.Namespace())
	assert.Equal(t, "v0.4.0-0", status.Version())
}
