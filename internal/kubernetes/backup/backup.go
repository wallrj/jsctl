package backup

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	v1alpha1kmsissuer "github.com/Skyscanner/kms-issuer/apis/certmanager/v1alpha1"
	v1alpha1approverpolicy "github.com/cert-manager/approver-policy/pkg/apis/policy/v1alpha1"
	v1beta1awspcaissuer "github.com/cert-manager/aws-privateca-issuer/pkg/api/v1beta1"
	v1certmanager "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	v1origincaissuer "github.com/cloudflare/origin-ca-issuer/pkgs/apis/v1"
	v1beta1googlecasissuer "github.com/jetstack/google-cas-issuer/api/v1beta1"
	v1alpha1vei "github.com/jetstack/venafi-enhanced-issuer/api/v1alpha1"
	v1beta1stepissuer "github.com/smallstep/step-issuer/api/v1beta1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/yaml"

	"github.com/jetstack/jsctl/internal/kubernetes/clients"
)

// ClusterBackupOptions wraps the options for fetching a cluster backup, sometimes
// not all resources are required, so these options allow for filtering and
// formatting.
type ClusterBackupOptions struct {
	RestConfig *rest.Config

	// FormatResources, if set, will remove certain fields from the resources
	FormatResources bool

	IncludeCertificates               bool
	IncludeIngressCertificates        bool
	IncludeIssuers                    bool
	IncludeCertificateRequestPolicies bool
}

type ClusterBackup []interface{}

func (c *ClusterBackup) ToYAML() ([]byte, error) {
	buf := new(bytes.Buffer)

	for _, r := range *c {
		buf.Write([]byte("---\n"))
		yamlBytes, err := yaml.Marshal(r)
		if err != nil {
			return nil, err
		}
		buf.Write(yamlBytes)
	}

	return []byte(strings.TrimSpace(buf.String()) + "\n"), nil
}

func FetchClusterBackup(ctx context.Context, opts ClusterBackupOptions) (*ClusterBackup, error) {
	var clusterBackup ClusterBackup

	// these fields should be excluded from the backup
	var dropFields []string
	if opts.FormatResources {
		dropFields = []string{
			"/metadata/creationTimestamp", // this is set as null in marshalling, but we clear the value anyway
			"/metadata/generation",
			"/metadata/resourceVersion",
			"/metadata/uid",
			"/metadata/managedFields",
			"/status",
			"/metadata/annotations/kubectl.kubernetes.io~1last-applied-configuration",
		}
	}

	// fetch all configured issuers and external issuers
	issuers, err := fetchAllIssuers(ctx, opts.RestConfig, dropFields)
	if err != nil {
		return &ClusterBackup{}, fmt.Errorf("failed to backup issuers: %w", err)
	}
	clusterBackup = append(clusterBackup, issuers...)

	// fetch certifcates
	certificateClient, err := clients.NewCertificateClient(opts.RestConfig)
	if err != nil {
		return &ClusterBackup{}, fmt.Errorf("failed to create client for certificates: %w", err)
	}

	var certificates v1certmanager.CertificateList
	err = certificateClient.List(
		ctx,
		&clients.GenericRequestOptions{DropFields: dropFields},
		&certificates,
	)
	if err != nil {
		return &ClusterBackup{}, fmt.Errorf("failed to list certificates: %w", err)
	}
	for _, c := range certificates.Items {
		// if we're not including ingress certificates, skip them
		skip := false
		if !opts.IncludeIngressCertificates && len(c.OwnerReferences) > 0 {
			for _, owner := range c.OwnerReferences {
				if owner.Kind == "Ingress" {
					skip = true
					break
				}
			}
		}
		if !skip {
			clusterBackup = append(clusterBackup, c)
		}
	}

	// fetch certificate request policies
	certificateRequestPolicyClient, err := clients.NewCertificateRequestPolicyClient(opts.RestConfig)
	if err != nil {
		return &ClusterBackup{}, fmt.Errorf("failed to create client for certificate request policies: %w", err)
	}

	var certificateRequestPolicies v1alpha1approverpolicy.CertificateRequestPolicyList
	err = certificateRequestPolicyClient.List(
		ctx,
		&clients.GenericRequestOptions{DropFields: dropFields},
		&certificateRequestPolicies,
	)
	if err != nil {
		return &ClusterBackup{}, fmt.Errorf("failed to list certificate request policies: %w", err)
	}
	for _, p := range certificateRequestPolicies.Items {
		clusterBackup = append(clusterBackup, p)
	}

	return &clusterBackup, nil
}

// TODO: this is similar to the logic in status.go, however with the types it's
// a pain to share functionality.
func fetchAllIssuers(ctx context.Context, cfg *rest.Config, dropFields []string) ([]interface{}, error) {
	requestOptions := &clients.GenericRequestOptions{
		DropFields: dropFields,
	}

	issuerClient, err := clients.NewAllIssuers(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create issuer client: %s", err)
	}
	issuerKinds, err := issuerClient.ListKinds(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list issuer kinds: %s", err)
	}

	var allIssuers []interface{}
	for _, kind := range issuerKinds {
		switch kind {
		case clients.CertManagerIssuer:
			client, err := clients.NewCertManagerIssuerClient(cfg)
			if err != nil {
				return nil, fmt.Errorf("failed to create clusterissuer client: %s", err)
			}
			var issuers v1certmanager.IssuerList
			err = client.List(ctx, requestOptions, &issuers)
			if err != nil {
				return nil, fmt.Errorf("failed to list clusterissuers: %s", err)
			}
			for _, issuer := range issuers.Items {
				allIssuers = append(allIssuers, issuer)
			}
		case clients.CertManagerClusterIssuer:
			client, err := clients.NewCertManagerClusterIssuerClient(cfg)
			if err != nil {
				return nil, fmt.Errorf("failed to create clusterissuer client: %s", err)
			}
			var clusterIssuers v1certmanager.ClusterIssuerList
			err = client.List(ctx, requestOptions, &clusterIssuers)
			if err != nil {
				return nil, fmt.Errorf("failed to list clusterissuers: %s", err)
			}
			for _, issuer := range clusterIssuers.Items {
				allIssuers = append(allIssuers, issuer)
			}
		case clients.GoogleCASIssuer:
			client, err := clients.NewGoogleCASIssuerClient(cfg)
			if err != nil {
				return nil, fmt.Errorf("failed to create cas client: %s", err)
			}
			var issuers v1beta1googlecasissuer.GoogleCASIssuerList
			err = client.List(ctx, requestOptions, &issuers)
			if err != nil {
				return nil, fmt.Errorf("failed to list cas issuers: %s", err)
			}
			for _, issuer := range issuers.Items {
				allIssuers = append(allIssuers, issuer)
			}
		case clients.GoogleCASClusterIssuer:
			client, err := clients.NewGoogleCASClusterIssuerClient(cfg)
			if err != nil {
				return nil, fmt.Errorf("failed to create cas cluster issuer client: %s", err)
			}
			var issuers v1beta1googlecasissuer.GoogleCASClusterIssuerList
			err = client.List(ctx, requestOptions, &issuers)
			if err != nil {
				return nil, fmt.Errorf("failed to list cas cluster issuers: %s", err)
			}
			for _, issuer := range issuers.Items {
				allIssuers = append(allIssuers, issuer)
			}
		case clients.AWSPCAIssuer:
			client, err := clients.NewAWSPCAIssuerClient(cfg)
			if err != nil {
				return nil, fmt.Errorf("failed to create aws pca issuer client: %s", err)
			}
			var issuers v1beta1awspcaissuer.AWSPCAIssuerList
			err = client.List(ctx, requestOptions, &issuers)
			if err != nil {
				return nil, fmt.Errorf("failed to list pca issuers: %s", err)
			}
			for _, issuer := range issuers.Items {
				allIssuers = append(allIssuers, issuer)
			}
		case clients.AWSPCAClusterIssuer:
			client, err := clients.NewAWSPCAClusterIssuerClient(cfg)
			if err != nil {
				return nil, fmt.Errorf("failed to create aws pca cluster issuer client: %s", err)
			}
			var issuers v1beta1awspcaissuer.AWSPCAClusterIssuerList
			err = client.List(ctx, requestOptions, &issuers)
			if err != nil {
				return nil, fmt.Errorf("failed to list pca cluster issuers: %s", err)
			}
			for _, issuer := range issuers.Items {
				allIssuers = append(allIssuers, issuer)
			}
		case clients.KMSIssuer:
			client, err := clients.NewKMSIssuerClient(cfg)
			if err != nil {
				return nil, fmt.Errorf("failed to create kms issuer client: %s", err)
			}
			var issuers v1alpha1kmsissuer.KMSIssuerList
			err = client.List(ctx, requestOptions, &issuers)
			if err != nil {
				return nil, fmt.Errorf("failed to list kms issuers: %s", err)
			}
			for _, issuer := range issuers.Items {
				allIssuers = append(allIssuers, issuer)
			}
		case clients.VenafiEnhancedIssuer:
			client, err := clients.NewVenafiEnhancedIssuerClient(cfg)
			if err != nil {
				return nil, fmt.Errorf("failed to create venafi enhanced issuer client: %s", err)
			}
			var issuers v1alpha1vei.VenafiIssuerList
			err = client.List(ctx, requestOptions, &issuers)
			if err != nil {
				return nil, fmt.Errorf("failed to list venafi issuers: %s", err)
			}
			for _, issuer := range issuers.Items {
				allIssuers = append(allIssuers, issuer)
			}
		case clients.VenafiEnhancedClusterIssuer:
			client, err := clients.NewVenafiEnhancedClusterIssuerClient(cfg)
			if err != nil {
				return nil, fmt.Errorf("failed to create venafi enhanced cluster issuer client: %s", err)
			}
			var issuers v1alpha1vei.VenafiClusterIssuerList
			err = client.List(ctx, requestOptions, &issuers)
			if err != nil {
				return nil, fmt.Errorf("failed to list venafi cluster issuers: %s", err)
			}
			for _, issuer := range issuers.Items {
				allIssuers = append(allIssuers, issuer)
			}
		case clients.OriginCAIssuer:
			client, err := clients.NewOriginCAIssuerClient(cfg)
			if err != nil {
				return nil, fmt.Errorf("failed to create origin ca issuer client: %s", err)
			}
			var issuers v1origincaissuer.OriginIssuerList
			err = client.List(ctx, requestOptions, &issuers)
			if err != nil {
				return nil, fmt.Errorf("failed to list origin ca issuers: %s", err)
			}
			for _, issuer := range issuers.Items {
				allIssuers = append(allIssuers, issuer)
			}
		case clients.SmallStepIssuer:
			client, err := clients.NewSmallStepIssuerClient(cfg)
			if err != nil {
				return nil, fmt.Errorf("failed to create smallstep issuer client: %s", err)
			}
			var issuers v1beta1stepissuer.StepIssuerList
			err = client.List(ctx, requestOptions, &issuers)
			if err != nil {
				return nil, fmt.Errorf("failed to list smallstep issuers: %s", err)
			}
			for _, issuer := range issuers.Items {
				allIssuers = append(allIssuers, issuer)
			}
		case clients.SmallStepClusterIssuer:
			client, err := clients.NewSmallStepClusterIssuerClient(cfg)
			if err != nil {
				return nil, fmt.Errorf("failed to create smallstep cluster issuer client: %s", err)
			}
			var issuers v1beta1stepissuer.StepClusterIssuerList
			err = client.List(ctx, requestOptions, &issuers)
			if err != nil {
				return nil, fmt.Errorf("failed to list smallstep cluster issuers: %s", err)
			}
			for _, issuer := range issuers.Items {
				allIssuers = append(allIssuers, issuer)
			}
		}
	}

	return allIssuers, nil
}
