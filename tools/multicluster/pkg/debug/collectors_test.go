package debug

import (
	"context"
	"testing"

	"github.com/10gen/ops-manager-kubernetes/multi/pkg/common"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/apps/v1"
	v12 "k8s.io/api/core/v1"
	v13 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	fake2 "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
)

func TestCollectors(t *testing.T) {
	ctx := context.Background()
	// given
	collectors := []Collector{
		&MongoDBCommunityCollector{},
		&MongoDBCollector{},
		&MongoDBMultiClusterCollector{},
		&MongoDBUserCollector{},
		&OpsManagerCollector{},
		&StatefulSetCollector{},
		&SecretCollector{},
		&ConfigMapCollector{},
		&RolesCollector{},
		&ServiceAccountCollector{},
		&RolesBindingsCollector{},
		&ServiceAccountCollector{},
	}
	filter := &AcceptAllFilter{}
	anonymizer := &NoOpAnonymizer{}
	namespace := "test"
	testObjectNames := "test"

	kubeClient := kubeClientWithTestingResources(ctx, namespace, testObjectNames)

	// when
	for _, collector := range collectors {
		kubeObjects, rawObjects, err := collector.Collect(ctx, kubeClient, namespace, filter, anonymizer)

		// then
		assert.NoError(t, err)
		assert.Equal(t, 1, len(kubeObjects))
		assert.Equal(t, 0, len(rawObjects))
	}
}

func kubeClientWithTestingResources(ctx context.Context, namespace, testObjectNames string) *common.KubeClientContainer {
	resources := []runtime.Object{
		&v12.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testObjectNames,
				Namespace: namespace,
			},
		},
		&v1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testObjectNames,
				Namespace: namespace,
			},
		},
		&v12.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testObjectNames,
				Namespace: namespace,
			},
		},
		&v12.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testObjectNames,
				Namespace: namespace,
			},
		},
		&v13.Role{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testObjectNames,
				Namespace: namespace,
			},
		},
		&v13.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testObjectNames,
				Namespace: namespace,
			},
		},
		&v12.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testObjectNames,
				Namespace: namespace,
			},
		},
	}

	// Unfortunately most of the Kind and Resource parts are guessing and making fake.NewSimpleDynamicClientWithCustomListKinds
	// happy. Sadly, it uses naming conventions (with List suffix) and tries to guess the plural names - mostly incorrectly.

	scheme := runtime.NewScheme()
	MongoDBCommunityGVK := schema.GroupVersionKind{
		Group:   MongoDBCommunityGVR.Group,
		Version: MongoDBCommunityGVR.Version,
		Kind:    "MongoDBCommunity",
	}
	MongoDBGVK := schema.GroupVersionKind{
		Group:   MongoDBGVR.Group,
		Version: MongoDBGVR.Version,
		Kind:    "MongoDB",
	}
	MongoDBUserGVK := schema.GroupVersionKind{
		Group:   MongoDBGVR.Group,
		Version: MongoDBGVR.Version,
		Kind:    "MongoDBUser",
	}
	MongoDBMultiGVK := schema.GroupVersionKind{
		Group:   MongoDBGVR.Group,
		Version: MongoDBGVR.Version,
		Kind:    "MongoDBMulti",
	}
	OpsManagerGVK := schema.GroupVersionKind{
		Group:   OpsManagerSchemeGVR.Group,
		Version: OpsManagerSchemeGVR.Version,
		Kind:    "OpsManager",
	}

	scheme.AddKnownTypeWithName(MongoDBCommunityGVK, &unstructured.Unstructured{})
	scheme.AddKnownTypeWithName(MongoDBGVK, &unstructured.Unstructured{})
	scheme.AddKnownTypeWithName(MongoDBMultiGVK, &unstructured.Unstructured{})
	scheme.AddKnownTypeWithName(MongoDBUserGVK, &unstructured.Unstructured{})

	MongoDBCommunityResource := unstructured.Unstructured{}
	MongoDBCommunityResource.SetGroupVersionKind(MongoDBCommunityGVK)
	MongoDBCommunityResource.SetName(testObjectNames)

	MongoDBResource := unstructured.Unstructured{}
	MongoDBResource.SetGroupVersionKind(MongoDBGVK)
	MongoDBResource.SetName(testObjectNames)

	MongoDBUserResource := unstructured.Unstructured{}
	MongoDBUserResource.SetGroupVersionKind(MongoDBUserGVK)
	MongoDBUserResource.SetName(testObjectNames)

	MongoDBMultiClusterResource := unstructured.Unstructured{}
	MongoDBMultiClusterResource.SetGroupVersionKind(MongoDBMultiGVK)
	MongoDBMultiClusterResource.SetName(testObjectNames)

	OpsManagerResource := unstructured.Unstructured{}
	OpsManagerResource.SetGroupVersionKind(OpsManagerGVK)
	OpsManagerResource.SetName(testObjectNames)

	dynamicLists := map[schema.GroupVersionResource]string{
		MongoDBCommunityGVR:    "MongoDBCommunityList",
		MongoDBGVR:             "MongoDBList",
		MongoDBUsersGVR:        "MongoDBUserList",
		MongoDBMultiClusterGVR: "MongoDBMultiClusterList",
		OpsManagerSchemeGVR:    "OpsManagerList",
	}
	dynamicFake := fake2.NewSimpleDynamicClientWithCustomListKinds(scheme, dynamicLists)

	dynamicFake.Resource(MongoDBMultiClusterGVR).Create(ctx, &MongoDBMultiClusterResource, metav1.CreateOptions{})
	dynamicFake.Resource(MongoDBCommunityGVR).Create(ctx, &MongoDBCommunityResource, metav1.CreateOptions{})
	dynamicFake.Resource(MongoDBGVR).Create(ctx, &MongoDBResource, metav1.CreateOptions{})
	dynamicFake.Resource(MongoDBUsersGVR).Create(ctx, &MongoDBUserResource, metav1.CreateOptions{})
	dynamicFake.Resource(OpsManagerSchemeGVR).Create(ctx, &OpsManagerResource, metav1.CreateOptions{})

	kubeClient := common.NewKubeClientContainer(nil, fake.NewSimpleClientset(resources...), dynamicFake)
	return kubeClient
}
