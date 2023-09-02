package kubernetes

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	apiv1 "k8s.io/api/core/v1"
	policy "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type Kubernetes interface {
	GetUnschedulableResources() (*UnschedulableResources, error)
	IsFailedSchedulingDueToControlPlaneTaint() (bool, error)
	GetKpNodes() ([]apiv1.Node, error)
	GetAllocatedResources() (map[string]*AllocatedResources, error)
	GetEmptyKpNodes() ([]apiv1.Node, error)
	CheckForNodeJoin(ctx context.Context, ok chan<- bool, newKpNodeName string)
	DeleteKpNode(kpNodeName string) error
	CordonKpNode(KpNodeName string) error
}

type KubernetesClient struct {
	client *kubernetes.Clientset
}

type UnschedulableResources struct {
	Cpu    float64
	Memory int64
}

type AllocatedResources struct {
	Cpu    float64
	Memory float64
}

func NewKubernetesClient() (KubernetesClient, error) {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
		flag.Parse()
	}

	var config *rest.Config

	if _, err := os.Stat(*kubeconfig); err == nil {
		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
		if err != nil {
			return KubernetesClient{}, err
		}
	} else {
		config, err = rest.InClusterConfig()
		if err != nil {
			panic(err.Error())
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	kubernetes := KubernetesClient{
		client: clientset,
	}

	return kubernetes, nil
}

func (k *KubernetesClient) GetUnschedulableResources() (*UnschedulableResources, error) {
	var rCpu float64
	var rMemory float64

	pods, err := k.client.CoreV1().Pods("").List(
		context.TODO(),
		metav1.ListOptions{},
	)
	if err != nil {
		return nil, err
	}

	for _, pod := range pods.Items {
		for _, condition := range pod.Status.Conditions {
			if condition.Type == apiv1.PodScheduled && condition.Status == apiv1.ConditionFalse && condition.Reason == "Unschedulable" {
				if strings.Contains(condition.Message, "Insufficient cpu") {
					for _, container := range pod.Spec.Containers {
						rCpu += container.Resources.Requests.Cpu().AsApproximateFloat64()
					}
				}
				if strings.Contains(condition.Message, "Insufficient memory") {
					for _, container := range pod.Spec.Containers {
						rMemory += container.Resources.Requests.Memory().AsApproximateFloat64()
					}
				}
			}
		}
	}

	unschedulableResources := &UnschedulableResources{
		Cpu:    rCpu,
		Memory: int64(rMemory),
	}

	return unschedulableResources, err
}

func (k *KubernetesClient) IsFailedSchedulingDueToControlPlaneTaint() (bool, error) {
	pods, err := k.client.CoreV1().Pods("").List(
		context.TODO(),
		metav1.ListOptions{},
	)
	if err != nil {
		return false, err
	}

	for _, pod := range pods.Items {
		for _, condition := range pod.Status.Conditions {
			if condition.Type == apiv1.PodScheduled && condition.Status == apiv1.ConditionFalse && condition.Reason == "Unschedulable" {
				if strings.Contains(condition.Message, "untolerated taint {node-role.kubernetes.io/control-plane:") {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

func (k *KubernetesClient) GetKpNodes() ([]apiv1.Node, error) {
	nodes, err := k.client.CoreV1().Nodes().List(
		context.TODO(),
		metav1.ListOptions{},
	)
	if err != nil {
		return nil, err
	}

	var kpNodes []apiv1.Node

	var kpNodeName = regexp.MustCompile(`^kp-node-\w{8}-\w{4}-\w{4}-\w{4}-\w{12}$`)

	for _, kpNode := range nodes.Items {
		if kpNodeName.MatchString(kpNode.Name) {
			kpNodes = append(kpNodes, kpNode)
		}
	}

	return kpNodes, err
}

func (k *KubernetesClient) GetAllocatedResources() (map[string]*AllocatedResources, error) {
	kpNodes, err := k.GetKpNodes()
	if err != nil {
		return nil, err
	}

	allocatedResources := map[string]*AllocatedResources{}

	for _, kpNode := range kpNodes {
		allocatedResources[kpNode.Name] = &AllocatedResources{
			Cpu:    0,
			Memory: 0,
		}

		pods, err := k.client.CoreV1().Pods("").List(
			context.TODO(),
			metav1.ListOptions{
				FieldSelector: fmt.Sprintf("spec.nodeName=%s", kpNode.Name),
			},
		)
		if err != nil {
			return nil, err
		}

		for _, pod := range pods.Items {
			for _, container := range pod.Spec.Containers {
				allocatedResources[kpNode.Name].Cpu += container.Resources.Requests.Cpu().AsApproximateFloat64()
				allocatedResources[kpNode.Name].Memory += container.Resources.Requests.Memory().AsApproximateFloat64()
			}
		}
	}

	return allocatedResources, err
}

func (k *KubernetesClient) GetEmptyKpNodes() ([]apiv1.Node, error) {
	nodes, err := k.GetKpNodes()
	if err != nil {
		return nil, err
	}

	var emptyNodes []apiv1.Node

	for _, node := range nodes {
		pods, err := k.client.CoreV1().Pods("").List(
			context.TODO(),
			metav1.ListOptions{
				FieldSelector: fmt.Sprintf("spec.nodeName=%s", node.Name),
			},
		)
		if err != nil {
			return nil, err
		}

		if len(pods.Items) == 0 {
			emptyNodes = append(emptyNodes, node)
		}
	}

	return emptyNodes, err
}

func (k *KubernetesClient) CheckForNodeJoin(ctx context.Context, ok chan<- bool, newKpNodeName string) {
	for {
		newkpNode, _ := k.client.CoreV1().Nodes().Get(
			context.TODO(),
			newKpNodeName,
			metav1.GetOptions{},
		)

		for _, condition := range newkpNode.Status.Conditions {
			if condition.Type == apiv1.NodeReady && condition.Status == apiv1.ConditionTrue {
				ok <- true
				return
			}
		}
	}
}

func (k *KubernetesClient) DeleteKpNode(kpNodeName string) error {
	err := k.CordonKpNode(kpNodeName)
	if err != nil {
		return err
	}

	pods, err := k.client.CoreV1().Pods("").List(
		context.TODO(),
		metav1.ListOptions{
			FieldSelector: fmt.Sprintf("spec.nodeName=%s", kpNodeName),
		},
	)
	if err != nil {
		return err
	}

	for _, pod := range pods.Items {
		k.client.PolicyV1().Evictions(pod.Namespace).Evict(
			context.TODO(),
			&policy.Eviction{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pod.Name,
					Namespace: pod.Namespace,
				},
			},
		)
	}

	err = k.client.CoreV1().Nodes().Delete(
		context.TODO(),
		kpNodeName,
		metav1.DeleteOptions{},
	)
	if err != nil {
		return err
	}

	return err
}

func (k *KubernetesClient) CordonKpNode(kpNodeName string) error {
	kpNode, err := k.client.CoreV1().Nodes().Get(
		context.TODO(),
		kpNodeName,
		metav1.GetOptions{},
	)
	if err != nil {
		return err
	}

	kpNode.Spec.Unschedulable = true

	_, err = k.client.CoreV1().Nodes().Update(
		context.TODO(),
		kpNode, metav1.UpdateOptions{},
	)

	return err
}
