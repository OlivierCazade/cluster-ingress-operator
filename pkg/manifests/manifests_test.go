package manifests

import (
	"fmt"
	"testing"
	"time"

	ingressv1alpha1 "github.com/openshift/cluster-ingress-operator/pkg/apis/ingress/v1alpha1"
	"github.com/openshift/cluster-ingress-operator/pkg/operator"
	"github.com/openshift/cluster-ingress-operator/pkg/util"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestManifests(t *testing.T) {
	config := operator.Config{RouterImage: "quay.io/openshift/router:latest"}
	f := NewFactory(config)

	ci := &ingressv1alpha1.ClusterIngress{
		ObjectMeta: metav1.ObjectMeta{
			Name: "default",
		},
		Spec: ingressv1alpha1.ClusterIngressSpec{
			NamespaceSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"foo": "bar",
				},
			},
			RouteSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"baz": "quux",
				},
			},
		},
	}

	if _, err := f.RouterNamespace(); err != nil {
		t.Errorf("invalid RouterNamespace: %v", err)
	}

	if _, err := f.RouterServiceAccount(); err != nil {
		t.Errorf("invalid RouterServiceAccount: %v", err)
	}

	if _, err := f.RouterClusterRole(); err != nil {
		t.Errorf("invalid RouterClusterRole: %v", err)
	}

	if _, err := f.RouterClusterRoleBinding(); err != nil {
		t.Errorf("invalid RouterClusterRoleBinding: %v", err)
	}

	deployment, err := f.RouterDeployment(ci)
	if err != nil {
		t.Errorf("invalid router Deployment: %v", err)
	}

	namespaceSelector := ""
	for _, envVar := range deployment.Spec.Template.Spec.Containers[0].Env {
		if envVar.Name == "NAMESPACE_LABELS" {
			namespaceSelector = envVar.Value
			break
		}
	}
	if namespaceSelector == "" {
		t.Error("router Deployment has no namespace selector")
	} else if namespaceSelector != "foo=bar" {
		t.Errorf("router Deployment has unexpected namespace selectors: %v",
			namespaceSelector)
	}

	routeSelector := ""
	for _, envVar := range deployment.Spec.Template.Spec.Containers[0].Env {
		if envVar.Name == "ROUTE_LABELS" {
			routeSelector = envVar.Value
			break
		}
	}
	if routeSelector == "" {
		t.Error("router Deployment has no route selector")
	} else if routeSelector != "baz=quux" {
		t.Errorf("router Deployment has unexpected route selectors: %v",
			routeSelector)
	}

	if len(deployment.Spec.Template.Spec.NodeSelector) == 0 {
		t.Error("router Deployment has no default node selector")
	}

	if deployment.Spec.Template.Spec.Volumes[0].Secret == nil {
		t.Error("router Deployment has no secret volume")
	}

	defaultSecretName := fmt.Sprintf("router-certs-%s", ci.Name)
	if deployment.Spec.Template.Spec.Volumes[0].Secret.SecretName != defaultSecretName {
		t.Errorf("router Deployment expected volume with secret %s, got %s",
			defaultSecretName, deployment.Spec.Template.Spec.Volumes[0].Secret.SecretName)
	}

	if svc, err := f.RouterServiceInternal(ci); err != nil {
		t.Errorf("invalid RouterServiceInternal: %v", err)
	} else if svc.Annotations[ServingCertSecretAnnotation] != defaultSecretName {
		t.Errorf("RouterServiceInternal expected serving secret annotation %s, got %s",
			defaultSecretName, svc.Annotations[ServingCertSecretAnnotation])
	}

	secretName := fmt.Sprintf("secret-%v", time.Now().UnixNano())
	ci.Spec.DefaultCertificateSecret = &secretName
	ci.Spec.NodePlacement = &ingressv1alpha1.NodePlacement{
		NodeSelector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"xyzzy": "quux",
			},
		},
	}
	deployment, err = f.RouterDeployment(ci)
	if err != nil {
		t.Errorf("invalid router Deployment: %v", err)
	}
	if len(deployment.Spec.Template.Spec.NodeSelector) != 1 ||
		deployment.Spec.Template.Spec.NodeSelector["xyzzy"] != "quux" {
		t.Errorf("router Deployment has unexpected node selector: %#v",
			deployment.Spec.Template.Spec.NodeSelector)
	}
	if e, a := config.RouterImage, deployment.Spec.Template.Spec.Containers[0].Image; e != a {
		t.Errorf("expected router Deployment image %q, got %q", e, a)
	}

	if deployment.Spec.Template.Spec.Volumes[0].Secret == nil {
		t.Error("router Deployment has no secret volume")
	}
	if deployment.Spec.Template.Spec.Volumes[0].Secret.SecretName != secretName {
		t.Errorf("expected router Deployment volume with secret %s, got %s",
			secretName, deployment.Spec.Template.Spec.Volumes[0].Secret.SecretName)
	}

	if svc, err := f.RouterServiceInternal(ci); err != nil {
		t.Errorf("invalid RouterServiceInternal: %v", err)
	} else if svc.Annotations[ServingCertSecretAnnotation] != defaultSecretName {
		t.Errorf("RouterServiceInternal expected serving secret annotation %s, got %s",
			defaultSecretName, svc.Annotations[ServingCertSecretAnnotation])
	}

	if _, err := f.RouterServiceCloud(ci); err != nil {
		t.Errorf("invalid RouterServiceCloud: %v", err)
	}
}

func TestDefaultClusterIngress(t *testing.T) {
	ic := &util.InstallConfig{
		Metadata: util.InstallConfigMetadata{
			Name: "user",
		},
		BaseDomain: "cluster.openshift.com",
	}
	def, err := NewFactory(operator.Config{RouterImage: "test"}).DefaultClusterIngress(ic)
	if err != nil {
		t.Fatal(err)
	}
	if e, a := "user.cluster.openshift.com", *def.Spec.IngressDomain; e != a {
		t.Errorf("expected default clusteringress ingressDomain=%s, got %s", e, a)
	}
}
