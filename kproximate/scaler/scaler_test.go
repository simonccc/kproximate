package scaler

import (
	"fmt"
	"testing"

	"github.com/lupinelab/kproximate/config"
	"github.com/lupinelab/kproximate/kubernetes"
	kproxmox "github.com/lupinelab/kproximate/proxmox"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
)

func TestRequiredScaleEventsFor1CPU(t *testing.T) {
	unschedulableResources := kubernetes.UnschedulableResources{
		Cpu:    1.0,
		Memory: 0,
	}

	s := Scaler{
		Config: config.KproximateConfig{
			KpNodeCores:  2,
			KpNodeMemory: 2048,
			KpNodeTemplateConfig: kproxmox.VMConfig{
				Cores:  2,
				Memory: 2048,
			},
			MaxKpNodes: 3,
		},
	}

	currentEvents := 0

	requiredScaleEvents := s.RequiredScaleEvents(&unschedulableResources, currentEvents)

	if len(requiredScaleEvents) != 1 {
		t.Errorf("Expected exactly 1 scaleEvent, got: %d", len(requiredScaleEvents))
	}
}

func TestRequiredScaleEventsFor3CPU(t *testing.T) {
	unschedulableResources := kubernetes.UnschedulableResources{
		Cpu:    3.0,
		Memory: 0,
	}

	s := Scaler{
		Config: config.KproximateConfig{
			KpNodeCores:  2,
			KpNodeMemory: 2048,
			KpNodeTemplateConfig: kproxmox.VMConfig{
				Cores:  2,
				Memory: 2048,
			},
			MaxKpNodes: 3,
		},
	}

	currentEvents := 0

	requiredScaleEvents := s.RequiredScaleEvents(&unschedulableResources, currentEvents)

	if len(requiredScaleEvents) != 2 {
		t.Errorf("Expected exactly 2 scaleEvents, got: %d", len(requiredScaleEvents))
	}
}

func TestRequiredScaleEventsFor1024MBMemory(t *testing.T) {
	unschedulableResources := kubernetes.UnschedulableResources{
		Cpu:    0,
		Memory: 1073741824,
	}

	s := Scaler{
		Config: config.KproximateConfig{
			KpNodeCores:  2,
			KpNodeMemory: 2048,
			KpNodeTemplateConfig: kproxmox.VMConfig{
				Cores:  2,
				Memory: 2048,
			},
			MaxKpNodes: 3,
		},
	}

	currentEvents := 0

	requiredScaleEvents := s.RequiredScaleEvents(&unschedulableResources, currentEvents)

	if len(requiredScaleEvents) != 1 {
		t.Errorf("Expected exactly 1 scaleEvent, got: %d", len(requiredScaleEvents))
	}
}

func TestRequiredScaleEventsFor3072MBMemory(t *testing.T) {
	unschedulableResources := kubernetes.UnschedulableResources{
		Cpu:    0,
		Memory: 3221225472,
	}

	s := Scaler{
		Config: config.KproximateConfig{
			KpNodeCores:  2,
			KpNodeMemory: 2048,
			KpNodeTemplateConfig: kproxmox.VMConfig{
				Cores:  2,
				Memory: 2048,
			},
			MaxKpNodes: 3,
		},
	}

	currentEvents := 0

	requiredScaleEvents := s.RequiredScaleEvents(&unschedulableResources, currentEvents)

	if len(requiredScaleEvents) != 2 {
		t.Errorf("Expected exactly 2 scaleEvent, got: %d", len(requiredScaleEvents))
	}
}

func TestRequiredScaleEventsFor1CPU3072MBMemory(t *testing.T) {
	unschedulableResources := kubernetes.UnschedulableResources{
		Cpu:    1,
		Memory: 3221225472,
	}

	s := Scaler{
		Config: config.KproximateConfig{
			KpNodeCores:  2,
			KpNodeMemory: 2048,
			KpNodeTemplateConfig: kproxmox.VMConfig{
				Cores:  2,
				Memory: 2048,
			},
			MaxKpNodes: 3,
		},
	}

	currentEvents := 0

	requiredScaleEvents := s.RequiredScaleEvents(&unschedulableResources, currentEvents)

	if len(requiredScaleEvents) != 2 {
		t.Errorf("Expected exactly 2 scaleEvent, got: %d", len(requiredScaleEvents))
	}
}

func TestRequiredScaleEventsFor1CPU3072MBMemory1QueuedEvent(t *testing.T) {
	unschedulableResources := kubernetes.UnschedulableResources{
		Cpu:    1,
		Memory: 3221225472,
	}

	s := Scaler{
		Config: config.KproximateConfig{
			KpNodeCores:  2,
			KpNodeMemory: 2048,
			KpNodeTemplateConfig: kproxmox.VMConfig{
				Cores:  2,
				Memory: 2048,
			},
			MaxKpNodes: 3,
		},
	}

	currentEvents := 1

	requiredScaleEvents := s.RequiredScaleEvents(&unschedulableResources, currentEvents)

	if len(requiredScaleEvents) != 1 {
		t.Errorf("Expected exactly 1 scaleEvent, got: %d", len(requiredScaleEvents))
	}
}

func TestSelectTargetPHosts(t *testing.T) {
	s := Scaler{
		PCluster: &kproxmox.ProxmoxMockClient{},
	}

	newName1 := fmt.Sprintf("kp-node-%s", uuid.NewUUID())
	newName2 := fmt.Sprintf("kp-node-%s", uuid.NewUUID())
	newName3 := fmt.Sprintf("kp-node-%s", uuid.NewUUID())

	scaleEvents := []*ScaleEvent{
		{
			ScaleType:  1,
			KpNodeName: newName1,
		},
		{
			ScaleType:  1,
			KpNodeName: newName2,
		},
		{
			ScaleType:  1,
			KpNodeName: newName3,
		},
	}

	s.SelectTargetPHosts(scaleEvents)

	if scaleEvents[0].TargetPHost.Id != "node/host-01" {
		t.Errorf("Expected node/host-01 to be selected as target pHost, got: %s", scaleEvents[0].TargetPHost.Id)
	}

	if scaleEvents[1].TargetPHost.Id != "node/host-02" {
		t.Errorf("Expected node/host-02 to be selected as target pHost, got: %s", scaleEvents[1].TargetPHost.Id)
	}

	if scaleEvents[2].TargetPHost.Id != "node/host-03" {
		t.Errorf("Expected node/host-03 to be selected as target pHost, got: %s", scaleEvents[2].TargetPHost.Id)
	}
}

func TestAssessScaleDownForResourceTypeZeroLoad(t *testing.T) {
	s := Scaler{
		Config: config.KproximateConfig{
			KpLoadHeadroom: 0.2,
		},
	}

	scaleDownZeroLoad := s.assessScaleDownForResourceType(0, 5, 5)
	if scaleDownZeroLoad == true {
		t.Errorf("Expected false but got %t", scaleDownZeroLoad)
	}
}

func TestAssessScaleDownForResourceTypeAcceptable(t *testing.T) {
	s := Scaler{
		Config: config.KproximateConfig{
			KpLoadHeadroom: 0.2,
		},
	}

	scaleDownAcceptable := s.assessScaleDownForResourceType(1, 5, 5)
	if scaleDownAcceptable != true {
		t.Errorf("Expected true but got %t", scaleDownAcceptable)
	}
}

func TestAssessScaleDownForResourceTypeUnAcceptable(t *testing.T) {
	s := Scaler{
		Config: config.KproximateConfig{
			KpLoadHeadroom: 0.2,
		},
	}
	scaleDownUnAcceptable := s.assessScaleDownForResourceType(4, 5, 5)
	if scaleDownUnAcceptable == true {
		t.Errorf("Expected false but got %t", scaleDownUnAcceptable)
	}
}

func TestSelectScaleDownTarget(t *testing.T) {
	s := Scaler{
		Config: config.KproximateConfig{
			KpNodeCores:  2,
			KpNodeMemory: 1024,
		},
	}

	scaleEvent := ScaleEvent{
		ScaleType: -1,
	}

	node1 := apiv1.Node{}
	node1.Name = "kp-node-163c3d58-4c4d-426d-baef-e0c30ecb5fcd"
	node2 := apiv1.Node{}
	node2.Name = "kp-node-a4f77d63-a944-425d-a980-e7be925b8a6a"
	node3 := apiv1.Node{}
	node3.Name = "kp-node-67944692-1de7-4bd0-ac8c-de6dc178cb38"
	kpNodes := []apiv1.Node{
		node1,
		node2,
		node3,
	}

	allocatedResources := map[string]*kubernetes.AllocatedResources{
		"kp-node-163c3d58-4c4d-426d-baef-e0c30ecb5fcd": {
			Cpu:    1.0,
			Memory: 2048.0,
		},
		"kp-node-a4f77d63-a944-425d-a980-e7be925b8a6a": {
			Cpu:    1.0,
			Memory: 2048.0,
		},
		"kp-node-67944692-1de7-4bd0-ac8c-de6dc178cb38": {
			Cpu:    1.0,
			Memory: 1048.0,
		},
	}

	s.SelectScaleDownTarget(&scaleEvent, allocatedResources, kpNodes)

	if scaleEvent.KpNodeName != "kp-node-67944692-1de7-4bd0-ac8c-de6dc178cb38" {
		t.Errorf("kp-node-67944692-1de7-4bd0-ac8c-de6dc178cb38 but got %s", scaleEvent.KpNodeName)
	}
}

func TestAssessScaleDownIsAcceptable(t *testing.T) {
	s := Scaler{
		Config: config.KproximateConfig{
			KpNodeCores:  2,
			KpNodeMemory: 1024,
		},
	}

	allocatedResources := map[string]*kubernetes.AllocatedResources{
		"kp-node-163c3d58-4c4d-426d-baef-e0c30ecb5fcd": {
			Cpu:    1.0,
			Memory: 2048.0,
		},
		"kp-node-a4f77d63-a944-425d-a980-e7be925b8a6a": {
			Cpu:    1.0,
			Memory: 2048.0,
		},
		"kp-node-67944692-1de7-4bd0-ac8c-de6dc178cb38": {
			Cpu:    1.0,
			Memory: 1048.0,
		},
	}

	numKpNodes := len(allocatedResources)

	if s.AssessScaleDown(allocatedResources, numKpNodes) == nil {
		t.Errorf("AssessScaleDown returned nil")
	}
}

func TestAssessScaleDownIsUnacceptable(t *testing.T) {
	s := Scaler{
		Config: config.KproximateConfig{
			KpNodeCores:  2,
			KpNodeMemory: 2048,
		},
	}

	allocatedResources := map[string]*kubernetes.AllocatedResources{
		"kp-node-163c3d58-4c4d-426d-baef-e0c30ecb5fcd": {
			Cpu:    2.0,
			Memory: 2147483648.0,
		},
		"kp-node-a4f77d63-a944-425d-a980-e7be925b8a6a": {
			Cpu:    2.0,
			Memory: 2147483648.0,
		},
		"kp-node-67944692-1de7-4bd0-ac8c-de6dc178cb38": {
			Cpu:    2.0,
			Memory: 2147483648.0,
		},
		"kp-node-a3c5e4ef-4713-473f-b9f7-3abe413c38ff": {
			Cpu:    2.0,
			Memory: 2147483648.0,
		},
		"kp-node-96f665dd-21c3-4ce1-a1e4-c7717c5338a3": {
			Cpu:    0.0,
			Memory: 0.0,
		},
	}

	numKpNodes := len(allocatedResources)

	if s.AssessScaleDown(allocatedResources, numKpNodes) != nil {
		t.Errorf("AssessScaleDown did not return nil")
	}
}
