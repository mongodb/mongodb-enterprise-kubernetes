package common

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	admissionregistrationv1 "k8s.io/client-go/kubernetes/typed/admissionregistration/v1"
	admissionregistrationv1alpha1 "k8s.io/client-go/kubernetes/typed/admissionregistration/v1alpha1"
	admissionregistrationv1beta1 "k8s.io/client-go/kubernetes/typed/admissionregistration/v1beta1"
	apiserverinternalv1alpha1 "k8s.io/client-go/kubernetes/typed/apiserverinternal/v1alpha1"
	appsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	appsv1beta1 "k8s.io/client-go/kubernetes/typed/apps/v1beta1"
	appsv1beta2 "k8s.io/client-go/kubernetes/typed/apps/v1beta2"
	authenticationv1 "k8s.io/client-go/kubernetes/typed/authentication/v1"
	authenticationv1alpha1 "k8s.io/client-go/kubernetes/typed/authentication/v1alpha1"
	authenticationv1beta1 "k8s.io/client-go/kubernetes/typed/authentication/v1beta1"
	authorizationv1 "k8s.io/client-go/kubernetes/typed/authorization/v1"
	authorizationv1beta1 "k8s.io/client-go/kubernetes/typed/authorization/v1beta1"
	autoscalingv1 "k8s.io/client-go/kubernetes/typed/autoscaling/v1"
	autoscalingv2 "k8s.io/client-go/kubernetes/typed/autoscaling/v2"
	autoscalingv2beta1 "k8s.io/client-go/kubernetes/typed/autoscaling/v2beta1"
	autoscalingv2beta2 "k8s.io/client-go/kubernetes/typed/autoscaling/v2beta2"
	batchv1 "k8s.io/client-go/kubernetes/typed/batch/v1"
	batchv1beta1 "k8s.io/client-go/kubernetes/typed/batch/v1beta1"
	certificatesv1 "k8s.io/client-go/kubernetes/typed/certificates/v1"
	certificatesv1alpha1 "k8s.io/client-go/kubernetes/typed/certificates/v1alpha1"
	certificatesv1beta1 "k8s.io/client-go/kubernetes/typed/certificates/v1beta1"
	coordinationv1 "k8s.io/client-go/kubernetes/typed/coordination/v1"
	coordinationv1beta1 "k8s.io/client-go/kubernetes/typed/coordination/v1beta1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	discoveryv1 "k8s.io/client-go/kubernetes/typed/discovery/v1"
	discoveryv1beta1 "k8s.io/client-go/kubernetes/typed/discovery/v1beta1"
	eventsv1 "k8s.io/client-go/kubernetes/typed/events/v1"
	eventsv1beta1 "k8s.io/client-go/kubernetes/typed/events/v1beta1"
	extensionsv1beta1 "k8s.io/client-go/kubernetes/typed/extensions/v1beta1"
	v1 "k8s.io/client-go/kubernetes/typed/flowcontrol/v1"
	flowcontrolv1beta1 "k8s.io/client-go/kubernetes/typed/flowcontrol/v1beta1"
	flowcontrolv1beta2 "k8s.io/client-go/kubernetes/typed/flowcontrol/v1beta2"
	flowcontrolv1beta3 "k8s.io/client-go/kubernetes/typed/flowcontrol/v1beta3"
	networkingv1 "k8s.io/client-go/kubernetes/typed/networking/v1"
	networkingv1alpha1 "k8s.io/client-go/kubernetes/typed/networking/v1alpha1"
	networkingv1beta1 "k8s.io/client-go/kubernetes/typed/networking/v1beta1"
	nodev1 "k8s.io/client-go/kubernetes/typed/node/v1"
	nodev1alpha1 "k8s.io/client-go/kubernetes/typed/node/v1alpha1"
	nodev1beta1 "k8s.io/client-go/kubernetes/typed/node/v1beta1"
	policyv1 "k8s.io/client-go/kubernetes/typed/policy/v1"
	policyv1beta1 "k8s.io/client-go/kubernetes/typed/policy/v1beta1"
	rbacv1 "k8s.io/client-go/kubernetes/typed/rbac/v1"
	rbacv1alpha1 "k8s.io/client-go/kubernetes/typed/rbac/v1alpha1"
	rbacv1beta1 "k8s.io/client-go/kubernetes/typed/rbac/v1beta1"
	resourcev1alpha2 "k8s.io/client-go/kubernetes/typed/resource/v1alpha2"
	schedulingv1 "k8s.io/client-go/kubernetes/typed/scheduling/v1"
	schedulingv1alpha1 "k8s.io/client-go/kubernetes/typed/scheduling/v1alpha1"
	schedulingv1beta1 "k8s.io/client-go/kubernetes/typed/scheduling/v1beta1"
	storagev1 "k8s.io/client-go/kubernetes/typed/storage/v1"
	storagev1alpha1 "k8s.io/client-go/kubernetes/typed/storage/v1alpha1"
	storagev1beta1 "k8s.io/client-go/kubernetes/typed/storage/v1beta1"
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

func (k *KubeClientContainer) FlowcontrolV1() v1.FlowcontrolV1Interface {
	panic("implement me")
}

func (k *KubeClientContainer) CertificatesV1alpha1() certificatesv1alpha1.CertificatesV1alpha1Interface {
	// TODO implement me
	panic("implement me")
}

func (k *KubeClientContainer) ResourceV1alpha2() resourcev1alpha2.ResourceV1alpha2Interface {
	// TODO implement me
	panic("implement me")
}

func (k *KubeClientContainer) AdmissionregistrationV1alpha1() admissionregistrationv1alpha1.AdmissionregistrationV1alpha1Interface {
	return k.staticClient.AdmissionregistrationV1alpha1()
}

func (k *KubeClientContainer) AuthenticationV1alpha1() authenticationv1alpha1.AuthenticationV1alpha1Interface {
	return k.staticClient.AuthenticationV1alpha1()
}

func (k *KubeClientContainer) FlowcontrolV1beta3() flowcontrolv1beta3.FlowcontrolV1beta3Interface {
	return k.staticClient.FlowcontrolV1beta3()
}

func (k *KubeClientContainer) NetworkingV1alpha1() networkingv1alpha1.NetworkingV1alpha1Interface {
	return k.staticClient.NetworkingV1alpha1()
}

func (k *KubeClientContainer) Discovery() discovery.DiscoveryInterface {
	return k.staticClient.Discovery()
}

func (k *KubeClientContainer) AdmissionregistrationV1() admissionregistrationv1.AdmissionregistrationV1Interface {
	return k.staticClient.AdmissionregistrationV1()
}

func (k *KubeClientContainer) AdmissionregistrationV1beta1() admissionregistrationv1beta1.AdmissionregistrationV1beta1Interface {
	return k.staticClient.AdmissionregistrationV1beta1()
}

func (k *KubeClientContainer) InternalV1alpha1() apiserverinternalv1alpha1.InternalV1alpha1Interface {
	return k.staticClient.InternalV1alpha1()
}

func (k *KubeClientContainer) AppsV1() appsv1.AppsV1Interface {
	return k.staticClient.AppsV1()
}

func (k *KubeClientContainer) AppsV1beta1() appsv1beta1.AppsV1beta1Interface {
	return k.staticClient.AppsV1beta1()
}

func (k *KubeClientContainer) AppsV1beta2() appsv1beta2.AppsV1beta2Interface {
	return k.staticClient.AppsV1beta2()
}

func (k *KubeClientContainer) AuthenticationV1() authenticationv1.AuthenticationV1Interface {
	return k.staticClient.AuthenticationV1()
}

func (k *KubeClientContainer) AuthenticationV1beta1() authenticationv1beta1.AuthenticationV1beta1Interface {
	return k.staticClient.AuthenticationV1beta1()
}

func (k *KubeClientContainer) AuthorizationV1() authorizationv1.AuthorizationV1Interface {
	return k.staticClient.AuthorizationV1()
}

func (k *KubeClientContainer) AuthorizationV1beta1() authorizationv1beta1.AuthorizationV1beta1Interface {
	return k.staticClient.AuthorizationV1beta1()
}

func (k *KubeClientContainer) AutoscalingV1() autoscalingv1.AutoscalingV1Interface {
	return k.staticClient.AutoscalingV1()
}

func (k *KubeClientContainer) AutoscalingV2() autoscalingv2.AutoscalingV2Interface {
	return k.staticClient.AutoscalingV2()
}

func (k *KubeClientContainer) AutoscalingV2beta1() autoscalingv2beta1.AutoscalingV2beta1Interface {
	return k.staticClient.AutoscalingV2beta1()
}

func (k *KubeClientContainer) AutoscalingV2beta2() autoscalingv2beta2.AutoscalingV2beta2Interface {
	return k.staticClient.AutoscalingV2beta2()
}

func (k *KubeClientContainer) BatchV1() batchv1.BatchV1Interface {
	return k.staticClient.BatchV1()
}

func (k *KubeClientContainer) BatchV1beta1() batchv1beta1.BatchV1beta1Interface {
	// TODO implement me
	panic("implement me")
}

func (k *KubeClientContainer) CertificatesV1() certificatesv1.CertificatesV1Interface {
	return k.staticClient.CertificatesV1()
}

func (k *KubeClientContainer) CertificatesV1beta1() certificatesv1beta1.CertificatesV1beta1Interface {
	return k.staticClient.CertificatesV1beta1()
}

func (k *KubeClientContainer) CoordinationV1beta1() coordinationv1beta1.CoordinationV1beta1Interface {
	return k.staticClient.CoordinationV1beta1()
}

func (k *KubeClientContainer) CoordinationV1() coordinationv1.CoordinationV1Interface {
	return k.staticClient.CoordinationV1()
}

func (k *KubeClientContainer) CoreV1() corev1.CoreV1Interface {
	return k.staticClient.CoreV1()
}

func (k *KubeClientContainer) DiscoveryV1() discoveryv1.DiscoveryV1Interface {
	return k.staticClient.DiscoveryV1()
}

func (k *KubeClientContainer) DiscoveryV1beta1() discoveryv1beta1.DiscoveryV1beta1Interface {
	return k.staticClient.DiscoveryV1beta1()
}

func (k KubeClientContainer) EventsV1() eventsv1.EventsV1Interface {
	return k.staticClient.EventsV1()
}

func (k *KubeClientContainer) EventsV1beta1() eventsv1beta1.EventsV1beta1Interface {
	return k.staticClient.EventsV1beta1()
}

func (k *KubeClientContainer) ExtensionsV1beta1() extensionsv1beta1.ExtensionsV1beta1Interface {
	return k.staticClient.ExtensionsV1beta1()
}

func (k *KubeClientContainer) FlowcontrolV1beta1() flowcontrolv1beta1.FlowcontrolV1beta1Interface {
	return k.staticClient.FlowcontrolV1beta1()
}

func (k *KubeClientContainer) FlowcontrolV1beta2() flowcontrolv1beta2.FlowcontrolV1beta2Interface {
	return k.staticClient.FlowcontrolV1beta2()
}

func (k *KubeClientContainer) NetworkingV1() networkingv1.NetworkingV1Interface {
	return k.staticClient.NetworkingV1()
}

func (k *KubeClientContainer) NetworkingV1beta1() networkingv1beta1.NetworkingV1beta1Interface {
	return k.staticClient.NetworkingV1beta1()
}

func (k *KubeClientContainer) NodeV1() nodev1.NodeV1Interface {
	return k.staticClient.NodeV1()
}

func (k *KubeClientContainer) NodeV1alpha1() nodev1alpha1.NodeV1alpha1Interface {
	return k.staticClient.NodeV1alpha1()
}

func (k *KubeClientContainer) NodeV1beta1() nodev1beta1.NodeV1beta1Interface {
	return k.staticClient.NodeV1beta1()
}

func (k *KubeClientContainer) PolicyV1() policyv1.PolicyV1Interface {
	return k.staticClient.PolicyV1()
}

func (k *KubeClientContainer) PolicyV1beta1() policyv1beta1.PolicyV1beta1Interface {
	return k.staticClient.PolicyV1beta1()
}

func (k *KubeClientContainer) RbacV1() rbacv1.RbacV1Interface {
	return k.staticClient.RbacV1()
}

func (k *KubeClientContainer) RbacV1beta1() rbacv1beta1.RbacV1beta1Interface {
	return k.staticClient.RbacV1beta1()
}

func (k *KubeClientContainer) RbacV1alpha1() rbacv1alpha1.RbacV1alpha1Interface {
	return k.staticClient.RbacV1alpha1()
}

func (k *KubeClientContainer) SchedulingV1alpha1() schedulingv1alpha1.SchedulingV1alpha1Interface {
	return k.staticClient.SchedulingV1alpha1()
}

func (k *KubeClientContainer) SchedulingV1beta1() schedulingv1beta1.SchedulingV1beta1Interface {
	return k.staticClient.SchedulingV1beta1()
}

func (k *KubeClientContainer) SchedulingV1() schedulingv1.SchedulingV1Interface {
	return k.staticClient.SchedulingV1()
}

func (k *KubeClientContainer) StorageV1beta1() storagev1beta1.StorageV1beta1Interface {
	return k.staticClient.StorageV1beta1()
}

func (k *KubeClientContainer) StorageV1() storagev1.StorageV1Interface {
	return k.staticClient.StorageV1()
}

func (k *KubeClientContainer) StorageV1alpha1() storagev1alpha1.StorageV1alpha1Interface {
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
