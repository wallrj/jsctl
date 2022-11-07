package components

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
)

func TestCertManagerWebhook(t *testing.T) {
	var err error
	data, err := os.ReadFile("fixtures/cert-manager-webhook.json")
	require.NoError(t, err)

	var pod v1.Pod

	err = json.Unmarshal(data, &pod)
	require.NoError(t, err)

	status, err := FindCertManagerWebhook(&pod)
	require.NoError(t, err)
	require.NotNilf(t, status, "expected status to be not nil")

	assert.Equal(t, "cert-manager-webhook", status.Name())
	assert.Equal(t, "jetstack-secure", status.Namespace())
	assert.Equal(t, "v1.9.1", status.Version())
}
