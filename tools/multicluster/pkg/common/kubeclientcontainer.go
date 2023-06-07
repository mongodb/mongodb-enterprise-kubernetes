package common

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/kubernetes/typed/admissionregistration/v1"
	v1beta16 "k8s.io/client-go/kubernetes/typed/admissionregistration/v1beta1"
	"k8s.io/client-go/kubernetes/typed/apiserverinternal/v1alpha1"
	v12 "k8s.io/client-go/kubernetes/typed/apps/v1"
	v1beta17 "k8s.io/client-go/kubernetes/typed/apps/v1beta1"
	"k8s.io/client-go/kubernetes/typed/apps/v1beta2"
	v17 "k8s.io/client-go/kubernetes/typed/authentication/v1"
	v1beta18 "k8s.io/client-go/kubernetes/typed/authentication/v1beta1"
	v18 "k8s.io/client-go/kubernetes/typed/authorization/v1"
	v1beta19 "k8s.io/client-go/kubernetes/typed/authorization/v1beta1"
	v19 "k8s.io/client-go/kubernetes/typed/autoscaling/v1"
	v2 "k8s.io/client-go/kubernetes/typed/autoscaling/v2"
	"k8s.io/client-go/kubernetes/typed/autoscaling/v2beta1"
	"k8s.io/client-go/kubernetes/typed/autoscaling/v2beta2"
	v110 "k8s.io/client-go/kubernetes/typed/batch/v1"
	"k8s.io/client-go/kubernetes/typed/batch/v1beta1"
	v111 "k8s.io/client-go/kubernetes/typed/certificates/v1"
	v1beta110 "k8s.io/client-go/kubernetes/typed/certificates/v1beta1"
	v116 "k8s.io/client-go/kubernetes/typed/coordination/v1"
	v1beta111 "k8s.io/client-go/kubernetes/typed/coordination/v1beta1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	v115 "k8s.io/client-go/kubernetes/typed/discovery/v1"
	v1beta117 "k8s.io/client-go/kubernetes/typed/discovery/v1beta1"
	v114 "k8s.io/client-go/kubernetes/typed/events/v1"
	v1beta116 "k8s.io/client-go/kubernetes/typed/events/v1beta1"
	v1beta115 "k8s.io/client-go/kubernetes/typed/extensions/v1beta1"
	v1alpha16 "k8s.io/client-go/kubernetes/typed/flowcontrol/v1alpha1"
	v1beta114 "k8s.io/client-go/kubernetes/typed/flowcontrol/v1beta1"
	v1beta22 "k8s.io/client-go/kubernetes/typed/flowcontrol/v1beta2"
	v113 "k8s.io/client-go/kubernetes/typed/networking/v1"
	v1beta113 "k8s.io/client-go/kubernetes/typed/networking/v1beta1"
	v112 "k8s.io/client-go/kubernetes/typed/node/v1"
	v1alpha15 "k8s.io/client-go/kubernetes/typed/node/v1alpha1"
	v1beta15 "k8s.io/client-go/kubernetes/typed/node/v1beta1"
	v16 "k8s.io/client-go/kubernetes/typed/policy/v1"
	v1beta14 "k8s.io/client-go/kubernetes/typed/policy/v1beta1"
	v15 "k8s.io/client-go/kubernetes/typed/rbac/v1"
	v1alpha13 "k8s.io/client-go/kubernetes/typed/rbac/v1alpha1"
	v1beta112 "k8s.io/client-go/kubernetes/typed/rbac/v1beta1"
	v14 "k8s.io/client-go/kubernetes/typed/scheduling/v1"
	v1alpha14 "k8s.io/client-go/kubernetes/typed/scheduling/v1alpha1"
	v1beta13 "k8s.io/client-go/kubernetes/typed/scheduling/v1beta1"
	v13 "k8s.io/client-go/kubernetes/typed/storage/v1"
	v1alpha12 "k8s.io/client-go/kubernetes/typed/storage/v1alpha1"
	v1beta12 "k8s.io/client-go/kubernetes/typed/storage/v1beta1"
	"k8s.io/client-go/rest"
)

// KubeClient is wrapper (decorator pattern) over the static and dynamic Kube Clients.
// It provides capabilities of both interfaces along with access to the initial REST configuration.
type KubeClient interface {
	kubernetes.Interface
	dynamic.Interface
	GetRestConfig() *rest.Config
}

var _ KubeClient = &KubeClientContainer{}

type KubeClientContainer struct {
	staticClient  kubernetes.Interface
	dynamicClient dynamic.Interface
	restConfig    *rest.Config
}

func (k *KubeClientContainer) Discovery() discovery.DiscoveryInterface {
	return k.staticClient.Discovery()
}

func (k *KubeClientContainer) AdmissionregistrationV1() v1.AdmissionregistrationV1Interface {
	return k.staticClient.AdmissionregistrationV1()
}

func (k *KubeClientContainer) AdmissionregistrationV1beta1() v1beta16.AdmissionregistrationV1beta1Interface {
	return k.staticClient.AdmissionregistrationV1beta1()
}

func (k *KubeClientContainer) InternalV1alpha1() v1alpha1.InternalV1alpha1Interface {
	return k.staticClient.InternalV1alpha1()
}

func (k *KubeClientContainer) AppsV1() v12.AppsV1Interface {
	return k.staticClient.AppsV1()
}

func (k *KubeClientContainer) AppsV1beta1() v1beta17.AppsV1beta1Interface {
	return k.staticClient.AppsV1beta1()
}

func (k *KubeClientContainer) AppsV1beta2() v1beta2.AppsV1beta2Interface {
	return k.staticClient.AppsV1beta2()
}

func (k *KubeClientContainer) AuthenticationV1() v17.AuthenticationV1Interface {
	return k.staticClient.AuthenticationV1()
}

func (k *KubeClientContainer) AuthenticationV1beta1() v1beta18.AuthenticationV1beta1Interface {
	return k.staticClient.AuthenticationV1beta1()
}

func (k *KubeClientContainer) AuthorizationV1() v18.AuthorizationV1Interface {
	return k.staticClient.AuthorizationV1()
}

func (k *KubeClientContainer) AuthorizationV1beta1() v1beta19.AuthorizationV1beta1Interface {
	return k.staticClient.AuthorizationV1beta1()
}

func (k *KubeClientContainer) AutoscalingV1() v19.AutoscalingV1Interface {
	return k.staticClient.AutoscalingV1()
}

func (k *KubeClientContainer) AutoscalingV2() v2.AutoscalingV2Interface {
	return k.staticClient.AutoscalingV2()
}

func (k *KubeClientContainer) AutoscalingV2beta1() v2beta1.AutoscalingV2beta1Interface {
	return k.staticClient.AutoscalingV2beta1()
}

func (k *KubeClientContainer) AutoscalingV2beta2() v2beta2.AutoscalingV2beta2Interface {
	return k.staticClient.AutoscalingV2beta2()
}

func (k *KubeClientContainer) BatchV1() v110.BatchV1Interface {
	return k.staticClient.BatchV1()
}

func (k *KubeClientContainer) BatchV1beta1() v1beta1.BatchV1beta1Interface {
	//TODO implement me
	panic("implement me")
}

func (k *KubeClientContainer) CertificatesV1() v111.CertificatesV1Interface {
	return k.staticClient.CertificatesV1()
}

func (k *KubeClientContainer) CertificatesV1beta1() v1beta110.CertificatesV1beta1Interface {
	return k.staticClient.CertificatesV1beta1()
}

func (k *KubeClientContainer) CoordinationV1beta1() v1beta111.CoordinationV1beta1Interface {
	return k.staticClient.CoordinationV1beta1()
}

func (k *KubeClientContainer) CoordinationV1() v116.CoordinationV1Interface {
	return k.staticClient.CoordinationV1()
}

func (k *KubeClientContainer) CoreV1() corev1client.CoreV1Interface {
	return k.staticClient.CoreV1()
}

func (k *KubeClientContainer) DiscoveryV1() v115.DiscoveryV1Interface {
	return k.staticClient.DiscoveryV1()
}

func (k *KubeClientContainer) DiscoveryV1beta1() v1beta117.DiscoveryV1beta1Interface {
	return k.staticClient.DiscoveryV1beta1()
}

func (k KubeClientContainer) EventsV1() v114.EventsV1Interface {
	return k.staticClient.EventsV1()
}

func (k *KubeClientContainer) EventsV1beta1() v1beta116.EventsV1beta1Interface {
	return k.staticClient.EventsV1beta1()
}

func (k *KubeClientContainer) ExtensionsV1beta1() v1beta115.ExtensionsV1beta1Interface {
	return k.staticClient.ExtensionsV1beta1()
}

func (k *KubeClientContainer) FlowcontrolV1alpha1() v1alpha16.FlowcontrolV1alpha1Interface {
	return k.staticClient.FlowcontrolV1alpha1()
}

func (k *KubeClientContainer) FlowcontrolV1beta1() v1beta114.FlowcontrolV1beta1Interface {
	return k.staticClient.FlowcontrolV1beta1()
}

func (k *KubeClientContainer) FlowcontrolV1beta2() v1beta22.FlowcontrolV1beta2Interface {
	return k.staticClient.FlowcontrolV1beta2()
}

func (k *KubeClientContainer) NetworkingV1() v113.NetworkingV1Interface {
	return k.staticClient.NetworkingV1()
}

func (k *KubeClientContainer) NetworkingV1beta1() v1beta113.NetworkingV1beta1Interface {
	return k.staticClient.NetworkingV1beta1()
}

func (k *KubeClientContainer) NodeV1() v112.NodeV1Interface {
	return k.staticClient.NodeV1()
}

func (k *KubeClientContainer) NodeV1alpha1() v1alpha15.NodeV1alpha1Interface {
	return k.staticClient.NodeV1alpha1()
}

func (k *KubeClientContainer) NodeV1beta1() v1beta15.NodeV1beta1Interface {
	return k.staticClient.NodeV1beta1()
}

func (k *KubeClientContainer) PolicyV1() v16.PolicyV1Interface {
	return k.staticClient.PolicyV1()
}

func (k *KubeClientContainer) PolicyV1beta1() v1beta14.PolicyV1beta1Interface {
	return k.staticClient.PolicyV1beta1()
}

func (k *KubeClientContainer) RbacV1() v15.RbacV1Interface {
	return k.staticClient.RbacV1()
}

func (k *KubeClientContainer) RbacV1beta1() v1beta112.RbacV1beta1Interface {
	return k.staticClient.RbacV1beta1()
}

func (k *KubeClientContainer) RbacV1alpha1() v1alpha13.RbacV1alpha1Interface {
	return k.staticClient.RbacV1alpha1()
}

func (k *KubeClientContainer) SchedulingV1alpha1() v1alpha14.SchedulingV1alpha1Interface {
	return k.staticClient.SchedulingV1alpha1()
}

func (k *KubeClientContainer) SchedulingV1beta1() v1beta13.SchedulingV1beta1Interface {
	return k.staticClient.SchedulingV1beta1()
}

func (k *KubeClientContainer) SchedulingV1() v14.SchedulingV1Interface {
	return k.staticClient.SchedulingV1()
}

func (k *KubeClientContainer) StorageV1beta1() v1beta12.StorageV1beta1Interface {
	return k.staticClient.StorageV1beta1()
}

func (k *KubeClientContainer) StorageV1() v13.StorageV1Interface {
	return k.staticClient.StorageV1()
}

func (k *KubeClientContainer) StorageV1alpha1() v1alpha12.StorageV1alpha1Interface {
	return k.staticClient.StorageV1alpha1()
}

func (k *KubeClientContainer) Resource(resource schema.GroupVersionResource) dynamic.NamespaceableResourceInterface {
	return k.dynamicClient.Resource(resource)
}

func (k *KubeClientContainer) GetRestConfig() *rest.Config {
	return k.restConfig
}

func NewKubeClientContainer(restConfig *rest.Config, staticClient kubernetes.Interface, dynamicClient dynamic.Interface) *KubeClientContainer {
	return &KubeClientContainer{
		staticClient:  staticClient,
		dynamicClient: dynamicClient,
		restConfig:    restConfig,
	}
}
