package status

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/rest"

	"github.com/jetstack/jsctl/internal/kubernetes/status/components"
)

func TestGatherClusterPreInstallStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		w.Header().Set("Content-Type", "application/json")
		// read the contents of fixtures files

		var data []byte
		switch r.URL.Path {
		case "/api/v1/namespaces":
			data, err = os.ReadFile("fixtures/namespace-list.json")
			require.NoError(t, err)
		case "/api/v1/pods":
			data, err = os.ReadFile("fixtures/pod-list.json")
			require.NoError(t, err)
		case "/apis/apiextensions.k8s.io/v1/customresourcedefinitions":
			data, err = os.ReadFile("fixtures/crd-list.json")
			require.NoError(t, err)
		case "/apis/networking.k8s.io/v1/ingresses":
			data, err = os.ReadFile("fixtures/ing-list.json")
			require.NoError(t, err)
		default:
			t.Fatalf("unexpected request: %s", r.URL.Path)
		}

		w.Write(data)
	}))

	cfg := &rest.Config{
		Host: server.URL,
	}

	status, err := GatherClusterPreInstallStatus(context.Background(), cfg)
	require.NoError(t, err)

	assert.Equal(t, status, &ClusterPreInstallStatus{
		Namepaces: []string{
			"jetstack-secure",
		},
		Ingresses: []summaryIngress{
			{
				Name:      "example",
				Namespace: "default",
				CertManagerAnnotations: map[string]string{
					"cert-manager.io/cluster-issuer": "nameOfClusterIssuer",
				},
			},
		},
		CRDGroups: []crdGroup{
			{
				Name: "cert-manager.io",
				CRDs: []string{
					"certificaterequestpolicies.policy.cert-manager.io",
					"certificaterequests.cert-manager.io",
					"certificates.cert-manager.io",
					"challenges.acme.cert-manager.io",
					"clusterissuers.cert-manager.io",
					"issuers.cert-manager.io",
					"orders.acme.cert-manager.io",
				},
			},
			{
				Name: "jetstack.io",
				CRDs: []string{
					"installations.operator.jetstack.io",
				},
			},
		},
		Components: map[string]installedComponent{
			"jetstack-secure-agent":        components.NewJetstackSeucreAgentStatus("jetstack-secure", "v0.1.38"),
			"jetstack-secure-operator":     components.NewJetstackSecureOperatorStatus("jetstack-secure", "v0.0.1-alpha.17"),
			"cert-manager-controller":      components.NewCertManagerControllerStatus("jetstack-secure", "v1.9.1"),
			"cert-manager-cainjector":      components.NewCertManagerCAInjectorStatus("jetstack-secure", "v1.9.1"),
			"cert-manager-webhook":         components.NewCertManagerWebhookStatus("jetstack-secure", "v1.9.1"),
			"cert-manager-approver-policy": components.NewCertManagerApproverPolicyStatus("jetstack-secure", "v0.4.0"),
		},
	})
}
