package debug

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"strings"

	"k8s.io/utils/ptr"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"

	"github.com/10gen/ops-manager-kubernetes/multi/pkg/common"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	// TODO: Report a bug on inconsistent naming (plural vs singular).
	MongoDBCommunityGVR    = schema.GroupVersionResource{Group: "mongodbcommunity.mongodb.com", Version: "v1", Resource: "mongodbcommunity"}
	MongoDBGVR             = schema.GroupVersionResource{Group: "mongodb.com", Version: "v1", Resource: "mongodb"}
	MongoDBMultiClusterGVR = schema.GroupVersionResource{Group: "mongodb.com", Version: "v1", Resource: "mongodbmulticlusters"}
	MongoDBUsersGVR        = schema.GroupVersionResource{Group: "mongodb.com", Version: "v1", Resource: "mongodbusers"}
	OpsManagerSchemeGVR    = schema.GroupVersionResource{Group: "mongodb.com", Version: "v1", Resource: "opsmanagers"}
)

const (
	redColor   = "\033[31m"
	resetColor = "\033[0m"
)

type Filter interface {
	Accept(object runtime.Object) bool
}

var _ Filter = &AcceptAllFilter{}

type AcceptAllFilter struct{}

func (a *AcceptAllFilter) Accept(_ runtime.Object) bool {
	return true
}

var _ Filter = &WithOwningReference{}

type WithOwningReference struct{}

func (a *WithOwningReference) Accept(object runtime.Object) bool {
	typeAccessor, err := meta.Accessor(object)
	if err != nil {
		return true
	}

	for _, or := range typeAccessor.GetOwnerReferences() {
		if strings.Contains(strings.ToLower(or.Kind), "mongo") {
			return true
		}
	}
	return false
}

type RawFile struct {
	Name          string
	ContainerName string
	content       []byte
}

type Collector interface {
	Collect(context.Context, common.KubeClient, string, Filter, Anonymizer) ([]runtime.Object, []RawFile, error)
}

var _ Collector = &StatefulSetCollector{}

type StatefulSetCollector struct{}

func (s *StatefulSetCollector) Collect(ctx context.Context, kubeClient common.KubeClient, namespace string, filter Filter, _ Anonymizer) ([]runtime.Object, []RawFile, error) {
	return genericCollect(ctx, kubeClient, namespace, filter, func(ctx context.Context, kubeClient common.KubeClient, namespace string) (runtime.Object, error) {
		return kubeClient.AppsV1().StatefulSets(namespace).List(ctx, v1.ListOptions{})
	})
}

var _ Collector = &ConfigMapCollector{}

type ConfigMapCollector struct{}

func (s *ConfigMapCollector) Collect(ctx context.Context, kubeClient common.KubeClient, namespace string, filter Filter, _ Anonymizer) ([]runtime.Object, []RawFile, error) {
	return genericCollect(ctx, kubeClient, namespace, filter, func(ctx context.Context, kubeClient common.KubeClient, namespace string) (runtime.Object, error) {
		return kubeClient.CoreV1().ConfigMaps(namespace).List(ctx, v1.ListOptions{})
	})
}

var _ Collector = &SecretCollector{}

type SecretCollector struct{}

func (s *SecretCollector) Collect(ctx context.Context, kubeClient common.KubeClient, namespace string, filter Filter, anonymizer Anonymizer) ([]runtime.Object, []RawFile, error) {
	var ret []runtime.Object
	secrets, err := kubeClient.CoreV1().Secrets(namespace).List(ctx, v1.ListOptions{})
	if err != nil {
		return nil, nil, err
	}
	for i := range secrets.Items {
		item := secrets.Items[i]
		if filter.Accept(&item) {
			ret = append(ret, anonymizer.AnonymizeSecret(&item))
		}
	}
	return ret, nil, nil
}

var _ Collector = &ServiceAccountCollector{}

type ServiceAccountCollector struct{}

func (s *ServiceAccountCollector) Collect(ctx context.Context, kubeClient common.KubeClient, namespace string, filter Filter, anonymizer Anonymizer) ([]runtime.Object, []RawFile, error) {
	return genericCollect(ctx, kubeClient, namespace, filter, func(ctx context.Context, kubeClient common.KubeClient, namespace string) (runtime.Object, error) {
		return kubeClient.CoreV1().ServiceAccounts(namespace).List(ctx, v1.ListOptions{})
	})
}

var _ Collector = &RolesCollector{}

type RolesCollector struct{}

func (s *RolesCollector) Collect(ctx context.Context, kubeClient common.KubeClient, namespace string, filter Filter, anonymizer Anonymizer) ([]runtime.Object, []RawFile, error) {
	return genericCollect(ctx, kubeClient, namespace, filter, func(ctx context.Context, kubeClient common.KubeClient, namespace string) (runtime.Object, error) {
		return kubeClient.RbacV1().Roles(namespace).List(ctx, v1.ListOptions{})
	})
}

var _ Collector = &RolesBindingsCollector{}

type RolesBindingsCollector struct{}

func (s *RolesBindingsCollector) Collect(ctx context.Context, kubeClient common.KubeClient, namespace string, filter Filter, anonymizer Anonymizer) ([]runtime.Object, []RawFile, error) {
	return genericCollect(ctx, kubeClient, namespace, filter, func(ctx context.Context, kubeClient common.KubeClient, namespace string) (runtime.Object, error) {
		return kubeClient.RbacV1().RoleBindings(namespace).List(ctx, v1.ListOptions{})
	})
}

var _ Collector = &MongoDBCollector{}

type MongoDBCollector struct{}

func (s *MongoDBCollector) Collect(ctx context.Context, kubeClient common.KubeClient, namespace string, filter Filter, anonymizer Anonymizer) ([]runtime.Object, []RawFile, error) {
	return genericCollect(ctx, kubeClient, namespace, filter, func(ctx context.Context, kubeClient common.KubeClient, namespace string) (runtime.Object, error) {
		return kubeClient.Resource(MongoDBGVR).List(ctx, v1.ListOptions{})
	})
}

var _ Collector = &MongoDBMultiClusterCollector{}

type MongoDBMultiClusterCollector struct{}

func (s *MongoDBMultiClusterCollector) Collect(ctx context.Context, kubeClient common.KubeClient, namespace string, filter Filter, anonymizer Anonymizer) ([]runtime.Object, []RawFile, error) {
	return genericCollect(ctx, kubeClient, namespace, filter, func(ctx context.Context, kubeClient common.KubeClient, namespace string) (runtime.Object, error) {
		return kubeClient.Resource(MongoDBMultiClusterGVR).List(ctx, v1.ListOptions{})
	})
}

var _ Collector = &MongoDBUserCollector{}

type MongoDBUserCollector struct{}

func (s *MongoDBUserCollector) Collect(ctx context.Context, kubeClient common.KubeClient, namespace string, filter Filter, anonymizer Anonymizer) ([]runtime.Object, []RawFile, error) {
	return genericCollect(ctx, kubeClient, namespace, filter, func(ctx context.Context, kubeClient common.KubeClient, namespace string) (runtime.Object, error) {
		return kubeClient.Resource(MongoDBUsersGVR).List(ctx, v1.ListOptions{})
	})
}

var _ Collector = &OpsManagerCollector{}

type OpsManagerCollector struct{}

func (s *OpsManagerCollector) Collect(ctx context.Context, kubeClient common.KubeClient, namespace string, filter Filter, anonymizer Anonymizer) ([]runtime.Object, []RawFile, error) {
	return genericCollect(ctx, kubeClient, namespace, filter, func(ctx context.Context, kubeClient common.KubeClient, namespace string) (runtime.Object, error) {
		return kubeClient.Resource(OpsManagerSchemeGVR).List(ctx, v1.ListOptions{})
	})
}

var _ Collector = &MongoDBCommunityCollector{}

type MongoDBCommunityCollector struct{}

func (s *MongoDBCommunityCollector) Collect(ctx context.Context, kubeClient common.KubeClient, namespace string, filter Filter, anonymizer Anonymizer) ([]runtime.Object, []RawFile, error) {
	return genericCollect(ctx, kubeClient, namespace, filter, func(ctx context.Context, kubeClient common.KubeClient, namespace string) (runtime.Object, error) {
		return kubeClient.Resource(MongoDBCommunityGVR).List(ctx, v1.ListOptions{})
	})
}

var _ Collector = &EventsCollector{}

type EventsCollector struct{}

func (s *EventsCollector) Collect(ctx context.Context, kubeClient common.KubeClient, namespace string, filter Filter, anonymizer Anonymizer) ([]runtime.Object, []RawFile, error) {
	return genericCollect(ctx, kubeClient, namespace, filter, func(ctx context.Context, kubeClient common.KubeClient, namespace string) (runtime.Object, error) {
		return kubeClient.EventsV1().Events(namespace).List(ctx, v1.ListOptions{})
	})
}

var _ Collector = &LogsCollector{}

type LogsCollector struct{}

func (s *LogsCollector) Collect(ctx context.Context, kubeClient common.KubeClient, namespace string, filter Filter, anonymizer Anonymizer) ([]runtime.Object, []RawFile, error) {
	pods, err := kubeClient.CoreV1().Pods(namespace).List(ctx, v1.ListOptions{})
	if err != nil {
		return nil, nil, err
	}
	var logsToCollect []RawFile
	for podIdx := range pods.Items {
		for containerIdx := range pods.Items[podIdx].Spec.Containers {
			logsToCollect = append(logsToCollect, RawFile{
				Name:          pods.Items[podIdx].Name,
				ContainerName: pods.Items[podIdx].Spec.Containers[containerIdx].Name,
			})
		}
	}
	for i := range logsToCollect {
		podName := logsToCollect[i].Name
		PodLogsConnection := kubeClient.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{
			Follow:    false,
			TailLines: ptr.To(int64(100)),
			Container: logsToCollect[i].ContainerName,
		})
		LogStream, err := PodLogsConnection.Stream(ctx)
		if err != nil {
			fmt.Printf(redColor+"[%T] error from %s/%s, ignoring: %s\n"+resetColor, s, namespace, podName, err)
			continue
		}
		reader := bufio.NewScanner(LogStream)
		var line string
		for reader.Scan() {
			line = fmt.Sprintf("%s\n", reader.Text())
			bytes := []byte(line)
			logsToCollect[i].content = append(logsToCollect[i].content, bytes...)
		}
		LogStream.Close()
	}
	return nil, logsToCollect, nil
}

var _ Collector = &AgentHealthFileCollector{}

type AgentHealthFileCollector struct{}

func (s *AgentHealthFileCollector) Collect(ctx context.Context, kubeClient common.KubeClient, namespace string, filter Filter, anonymizer Anonymizer) ([]runtime.Object, []RawFile, error) {
	type AgentHealthFileToCollect struct {
		podName       string
		RawFile       rest.ContentConfig
		agentFileName string
		containerName string
	}

	pods, err := kubeClient.CoreV1().Pods(namespace).List(ctx, v1.ListOptions{})
	if err != nil {
		return nil, nil, err
	}
	var logsToCollect []AgentHealthFileToCollect
	var collectedHealthFiles []RawFile
	for i, pod := range pods.Items {
		add := AgentHealthFileToCollect{
			podName: pods.Items[i].Name,
		}
		found := false
		for _, c := range pod.Spec.Containers {
			for _, e := range c.Env {
				if "AGENT_STATUS_FILEPATH" == e.Name {
					add.agentFileName = e.Value
					found = true
					break
				}
			}
			if found {
				add.containerName = c.Name
				break
			}
		}

		if found {
			logsToCollect = append(logsToCollect, add)
		}
	}
	for _, l := range logsToCollect {
		add := RawFile{
			Name: l.podName + "-agent-health",
		}
		content, err := getFileContent(kubeClient.GetRestConfig(), kubeClient, namespace, l.podName, l.containerName, l.agentFileName)
		if err == nil {
			add.content = content
			collectedHealthFiles = append(collectedHealthFiles, add)
		}
	}
	return nil, collectedHealthFiles, nil
}

// Inspired by https://gist.github.com/kyroy/8453a0c4e075e91809db9749e0adcff2
func getFileContent(config *rest.Config, clientset common.KubeClient, namespace, podName, containerName, path string) ([]byte, error) {
	u := clientset.CoreV1().RESTClient().Post().
		Namespace(namespace).
		Name(podName).
		Resource("pods").
		SubResource("exec").
		Param("command", "/bin/cat").
		Param("command", path).
		Param("container", containerName).
		Param("stderr", "true").
		Param("stdout", "true").URL()

	buf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	exec, err := remotecommand.NewSPDYExecutor(config, "POST", u)
	err = exec.Stream(remotecommand.StreamOptions{
		Stdout: buf,
		Stderr: errBuf,
	})
	if err != nil {
		return nil, fmt.Errorf("%w Failed obtaining file %s from %v/%v", err, path, namespace, podName)
	}

	return buf.Bytes(), nil
}

type genericLister func(ctx context.Context, kubeClient common.KubeClient, namespace string) (runtime.Object, error)

func genericCollect(ctx context.Context, kubeClient common.KubeClient, namespace string, filter Filter, lister genericLister) ([]runtime.Object, []RawFile, error) {
	var ret []runtime.Object
	listAsObject, err := lister(ctx, kubeClient, namespace)
	if err != nil {
		return nil, nil, err
	}
	list, err := meta.ExtractList(listAsObject)
	if err != nil {
		return nil, nil, err
	}
	for i := range list {
		item := list[i]
		if filter.Accept(item) {
			ret = append(ret, item)
		}
	}
	return ret, nil, nil
}

type CollectionResult struct {
	kubeResources []runtime.Object
	rawObjects    []RawFile
	errors        []error
	namespace     string
	context       string
}

func Collect(ctx context.Context, kubeClient common.KubeClient, context string, namespace string, filter Filter, collectors []Collector, anonymizer Anonymizer) CollectionResult {
	result := CollectionResult{}
	result.context = context
	result.namespace = namespace

	for _, collector := range collectors {
		collectedKubeObjects, collectedRawObjects, err := collector.Collect(ctx, kubeClient, namespace, filter, anonymizer)
		errorString := ""
		if err != nil {
			errorString = fmt.Sprintf(redColor+" error: %s"+resetColor, err)
		}
		fmt.Printf("[%T] collected %d kubeObjects, %d rawObjects%s\n", collector, len(collectedKubeObjects), len(collectedRawObjects), errorString)
		result.kubeResources = append(result.kubeResources, collectedKubeObjects...)
		result.rawObjects = append(result.rawObjects, collectedRawObjects...)
		if err != nil {
			result.errors = append(result.errors, err)
		}
	}
	return result
}
