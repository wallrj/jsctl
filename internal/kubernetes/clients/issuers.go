package clients

import (
	"context"
	"fmt"

	awspca "github.com/cert-manager/aws-privateca-issuer/pkg/api/v1beta1"
	cmapi "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	googlecas "github.com/jetstack/google-cas-issuer/api/v1beta1"
	v1extenstions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/client-go/rest"
)

// AnyIssuer is an enum of all known issuer types, external and built-in.
type AnyIssuer int64

const (
	CertManagerIssuer AnyIssuer = iota
	CertManagerClusterIssuer
	AWSPCAIssuer
	AWSPCAClusterIssuer
	KMSIssuer
	GoogleCASIssuer
	GoogleCASClusterIssuer
	OriginCAIssuer
	// OriginCAClusterIssuer TODO: confirm this doesn't have a cluster issuer
	SmallStepIssuer
	SmallStepClusterIssuer
	// KMSClusterIssuer TODO: confirm this doesn't have a cluster issuer
)

var AllIssuersList = []AnyIssuer{
	CertManagerIssuer,
	CertManagerClusterIssuer,
	AWSPCAIssuer,
	AWSPCAClusterIssuer,
	KMSIssuer,
	GoogleCASIssuer,
	GoogleCASClusterIssuer,
	OriginCAIssuer,
	SmallStepIssuer,
	SmallStepClusterIssuer,
}

func (s AnyIssuer) String() string {
	switch s {
	case CertManagerIssuer:
		return "issuers.cert-manager.io"
	case CertManagerClusterIssuer:
		return "clusterissuers.cert-manager.io"
	case AWSPCAIssuer:
		return "awspcaissuers.awspca.cert-manager.io"
	case AWSPCAClusterIssuer:
		return "awspcaclusterissuers.awspca.cert-manager.io"
	case KMSIssuer:
		return "kmsissuers.cert-manager.skyscanner.net"
	case GoogleCASIssuer:
		return "googlecasissuers.cas-issuer.jetstack.io"
	case GoogleCASClusterIssuer:
		return "googlecasclusterissuers.cas-issuer.jetstack.io"
	case OriginCAIssuer:
		return "originissuers.cert-manager.k8s.cloudflare.com"
	case SmallStepIssuer:
		return "stepissuers.certmanager.step.sm"
	case SmallStepClusterIssuer:
		return "stepclusterissuers.certmanager.step.sm"
	}
	return "unknown"
}

// AllIssuers is a special client to wrap logic for determining the kinds of
// issuers present in a cluster
type AllIssuers struct {
	crdClient *Generic[*v1extenstions.CustomResourceDefinition, *v1extenstions.CustomResourceDefinitionList]
}

func (a *AllIssuers) ListKinds(ctx context.Context) ([]AnyIssuer, error) {
	// form an index of all known issuer types
	issuerIndex := make(map[string]AnyIssuer)
	for _, issuer := range AllIssuersList {
		issuerIndex[issuer.String()] = issuer
	}

	var crds v1extenstions.CustomResourceDefinitionList
	err := a.crdClient.List(ctx, &GenericRequestOptions{}, &crds)

	if err != nil {
		return nil, fmt.Errorf("error listing CRDs: %w", err)
	}

	var foundIssuers []AnyIssuer
	for _, crd := range crds.Items {
		anyIssuer, ok := issuerIndex[crd.Name]
		if ok {
			foundIssuers = append(foundIssuers, anyIssuer)
		}
	}

	return foundIssuers, nil
}

// NewAllIssuers returns a new instance of and AllIssuers client.
func NewAllIssuers(config *rest.Config) (*AllIssuers, error) {
	crdClient, err := NewCRDClient(config)
	if err != nil {
		return nil, fmt.Errorf("error creating CRD client: %w", err)
	}

	return &AllIssuers{
		crdClient: crdClient,
	}, nil
}

// NewCertManagerIssuerClient returns an instance of a generic client for querying
// cert-manager Issuers
func NewCertManagerIssuerClient(config *rest.Config) (*Generic[*cmapi.Issuer, *cmapi.IssuerList], error) {
	genericClient, err := NewGenericClient[*cmapi.Issuer, *cmapi.IssuerList](
		&GenericClientOptions{
			RestConfig: config,
			APIPath:    "/apis",
			Group:      cmapi.SchemeGroupVersion.Group,
			Version:    cmapi.SchemeGroupVersion.Version,
			Kind:       "issuers",
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error creating generic client: %w", err)
	}

	return genericClient, nil
}

// NewCertManagerClusterIssuerClient returns an instance of a generic client
// for querying cert-manager ClusterIssuers
func NewCertManagerClusterIssuerClient(config *rest.Config) (*Generic[*cmapi.ClusterIssuer, *cmapi.ClusterIssuerList], error) {
	genericClient, err := NewGenericClient[*cmapi.ClusterIssuer, *cmapi.ClusterIssuerList](
		&GenericClientOptions{
			RestConfig: config,
			APIPath:    "/apis",
			Group:      cmapi.SchemeGroupVersion.Group,
			Version:    cmapi.SchemeGroupVersion.Version,
			Kind:       "clusterissuers",
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error creating generic client: %w", err)
	}

	return genericClient, nil
}

// NewGoogleCASIssuerClient returns an instance of a generic client for querying
// google CAS Issuers
func NewGoogleCASIssuerClient(config *rest.Config) (*Generic[*googlecas.GoogleCASIssuer, *googlecas.GoogleCASIssuerList], error) {
	genericClient, err := NewGenericClient[*googlecas.GoogleCASIssuer, *googlecas.GoogleCASIssuerList](
		&GenericClientOptions{
			RestConfig: config,
			APIPath:    "/apis",
			Group:      googlecas.GroupVersion.Group,
			Version:    googlecas.GroupVersion.Version,
			Kind:       "googlecasissuers",
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error creating generic client: %w", err)
	}

	return genericClient, nil
}

// NewGoogleCASClusterIssuerClient returns an instance of a generic client for querying
// google CAS cluster Issuers
func NewGoogleCASClusterIssuerClient(config *rest.Config) (*Generic[*googlecas.GoogleCASClusterIssuer, *googlecas.GoogleCASClusterIssuerList], error) {
	genericClient, err := NewGenericClient[*googlecas.GoogleCASClusterIssuer, *googlecas.GoogleCASClusterIssuerList](
		&GenericClientOptions{
			RestConfig: config,
			APIPath:    "/apis",
			Group:      googlecas.GroupVersion.Group,
			Version:    googlecas.GroupVersion.Version,
			Kind:       "googlecasclusterissuers",
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error creating generic client: %w", err)
	}

	return genericClient, nil
}

// NewAWSPCAIssuerClient returns an instance of a generic client for querying
// AWS PCA Issuers
func NewAWSPCAIssuerClient(config *rest.Config) (*Generic[*awspca.AWSPCAIssuer, *awspca.AWSPCAIssuerList], error) {
	genericClient, err := NewGenericClient[*awspca.AWSPCAIssuer, *awspca.AWSPCAIssuerList](
		&GenericClientOptions{
			RestConfig: config,
			APIPath:    "/apis",
			Group:      awspca.GroupVersion.Group,
			Version:    awspca.GroupVersion.Version,
			Kind:       "awspcaissuers",
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error creating generic client: %w", err)
	}

	return genericClient, nil
}

// NewAWSPCAClusterIssuerClient returns an instance of a generic client for querying
// AWS PCA cluster Issuers
func NewAWSPCAClusterIssuerClient(config *rest.Config) (*Generic[*awspca.AWSPCAClusterIssuer, *awspca.AWSPCAClusterIssuerList], error) {
	genericClient, err := NewGenericClient[*awspca.AWSPCAClusterIssuer, *awspca.AWSPCAClusterIssuerList](
		&GenericClientOptions{
			RestConfig: config,
			APIPath:    "/apis",
			Group:      awspca.GroupVersion.Group,
			Version:    awspca.GroupVersion.Version,
			Kind:       "awspcaclusterissuers",
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error creating generic client: %w", err)
	}

	return genericClient, nil
}
