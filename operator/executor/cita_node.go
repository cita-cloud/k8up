package executor

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	citav1 "github.com/k8up-io/k8up/v2/api/v1cita"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Creator func(namespace, name string, client client.Client, logger logr.Logger) (Node, error)

var nodes = make(map[citav1.DeployMethod]Creator)

func Register(deployMethod citav1.DeployMethod, register Creator) {
	nodes[deployMethod] = register
}

func CreateNode(deployMethod citav1.DeployMethod, namespace, name string, client client.Client, logger logr.Logger) (Node, error) {
	f, ok := nodes[deployMethod]
	if ok {
		return f(namespace, name, client, logger)
	}
	return nil, fmt.Errorf("invalid deploy type: %s", string(deployMethod))
}

type Node interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	CheckStopped(ctx context.Context) (bool, error)
	GetPVCInfo(ctx context.Context) (corev1.ResourceRequirements, error)
	GetVolume(ctx context.Context) (corev1.Volume, error)
	GetCryptoAndConsensus(ctx context.Context) (string, string, error)
}

type cloudConfigNode struct {
	client.Client
	logger    logr.Logger
	namespace string
	name      string
}

func (c *cloudConfigNode) GetCryptoAndConsensus(ctx context.Context) (string, string, error) {
	sts := &appsv1.StatefulSet{}
	err := c.Client.Get(ctx, types.NamespacedName{Namespace: c.namespace, Name: c.name}, sts)
	if err != nil {
		return "", "", err
	}

	var crypto, consensus string
	for _, container := range sts.Spec.Template.Spec.Containers {
		if container.Name == "crypto" || container.Name == "kms" {
			if strings.Contains(container.Image, "sm") {
				crypto = "sm"
			} else if strings.Contains(container.Image, "eth") {
				crypto = "eth"
			}
		} else if container.Name == "consensus" {
			if strings.Contains(container.Image, "bft") {
				consensus = "bft"
			} else if strings.Contains(container.Image, "raft") {
				consensus = "raft"
			} else if strings.Contains(container.Image, "overlord") {
				consensus = "overlord"
			}
		}
	}
	return crypto, consensus, nil
}

func (c *cloudConfigNode) GetVolume(ctx context.Context) (corev1.Volume, error) {
	return corev1.Volume{
		Name: "backup-source",
		VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: fmt.Sprintf("datadir-%s-0", c.name),
				ReadOnly:  false,
			},
		},
	}, nil
}

func (c *cloudConfigNode) GetPVCInfo(ctx context.Context) (corev1.ResourceRequirements, error) {
	var resourceRequirements corev1.ResourceRequirements
	sts := &appsv1.StatefulSet{}
	if err := c.Get(ctx, types.NamespacedName{Name: c.name, Namespace: c.namespace}, sts); err != nil {
		return resourceRequirements, err
	}
	pvcs := sts.Spec.VolumeClaimTemplates
	for _, pvc := range pvcs {
		if pvc.Name == "datadir" {
			resourceRequirements = pvc.Spec.Resources
			break
		}
	}
	return resourceRequirements, nil
}

func (c *cloudConfigNode) CheckStopped(ctx context.Context) (bool, error) {
	found := &appsv1.StatefulSet{}
	err := c.Client.Get(ctx, types.NamespacedName{Name: c.name, Namespace: c.namespace}, found)
	if err != nil {
		return false, err
	}
	if found.Status.ReadyReplicas == 0 {
		return true, nil
	}
	return false, nil
}

func (c *cloudConfigNode) Start(ctx context.Context) error {
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		sts := &appsv1.StatefulSet{}
		err := c.Client.Get(ctx, types.NamespacedName{Name: c.name, Namespace: c.namespace}, sts)
		if err != nil {
			return err
		}
		sts.Spec.Replicas = pointer.Int32(1)
		err = c.Client.Update(ctx, sts)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		c.logger.Error(err, "start node failed", "name", c.name, "namespace", c.namespace)
		return err
	}
	c.logger.Info("start node success", "name", c.name, "namespace", c.namespace)
	return nil
}

func (c *cloudConfigNode) Stop(ctx context.Context) error {
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		sts := &appsv1.StatefulSet{}
		err := c.Client.Get(ctx, types.NamespacedName{Name: c.name, Namespace: c.namespace}, sts)
		if err != nil {
			return err
		}
		sts.Spec.Replicas = pointer.Int32(0)
		err = c.Client.Update(ctx, sts)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		c.logger.Error(err, "stop node failed", "name", c.name, "namespace", c.namespace)
		return err
	}
	c.logger.Info("stop node success", "name", c.name, "namespace", c.namespace)
	return nil
}

func newCloudConfigNode(namespace, name string, client client.Client, logger logr.Logger) (Node, error) {
	return &cloudConfigNode{
		Client:    client,
		namespace: namespace,
		name:      name,
		logger:    logger,
	}, nil
}

type pyNode struct {
	client.Client
	logger    logr.Logger
	namespace string
	name      string
}

func (p *pyNode) GetCryptoAndConsensus(ctx context.Context) (string, string, error) {
	dep := &appsv1.Deployment{}
	err := p.Client.Get(ctx, types.NamespacedName{Namespace: p.namespace, Name: p.name}, dep)
	if err != nil {
		return "", "", err
	}

	var crypto, consensus string
	for _, container := range dep.Spec.Template.Spec.Containers {
		if container.Name == "crypto" || container.Name == "kms" {
			if strings.Contains(container.Image, "sm") {
				crypto = "sm"
			} else if strings.Contains(container.Image, "eth") {
				crypto = "eth"
			}
		} else if container.Name == "consensus" {
			if strings.Contains(container.Image, "bft") {
				consensus = "bft"
			} else if strings.Contains(container.Image, "raft") {
				consensus = "raft"
			} else if strings.Contains(container.Image, "overlord") {
				consensus = "overlord"
			}
		}
	}
	return crypto, consensus, nil
}

func (p *pyNode) GetVolume(ctx context.Context) (corev1.Volume, error) {
	var volume corev1.Volume
	var pvcName string
	deploy := &appsv1.Deployment{}
	if err := p.Get(ctx, types.NamespacedName{Name: p.name, Namespace: p.namespace}, deploy); err != nil {
		return volume, err
	}
	volumes := deploy.Spec.Template.Spec.Volumes
	for _, vol := range volumes {
		if vol.Name == "datadir" {
			pvcName = vol.PersistentVolumeClaim.ClaimName
			break
		}
	}
	return corev1.Volume{
		Name: "backup-source",
		VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: pvcName,
				ReadOnly:  false,
			},
		},
	}, nil
}

func (p *pyNode) GetPVCInfo(ctx context.Context) (corev1.ResourceRequirements, error) {
	var resourceRequirements corev1.ResourceRequirements
	var pvcSourceName string

	deploy := &appsv1.Deployment{}
	if err := p.Get(ctx, types.NamespacedName{Name: p.name, Namespace: p.namespace}, deploy); err != nil {
		return resourceRequirements, err
	}
	volumes := deploy.Spec.Template.Spec.Volumes
	for _, volume := range volumes {
		if volume.Name == "datadir" {
			pvcSourceName = volume.PersistentVolumeClaim.ClaimName
			break
		}
	}
	if pvcSourceName == "" {
		return resourceRequirements, fmt.Errorf("cann't get deployment's pvc")
	}
	pvc := &corev1.PersistentVolumeClaim{}
	err := p.Get(ctx, types.NamespacedName{Name: pvcSourceName, Namespace: p.namespace}, pvc)
	if err != nil {
		return resourceRequirements, err
	}
	return pvc.Spec.Resources, nil
}

func (p *pyNode) CheckStopped(ctx context.Context) (bool, error) {
	found := &appsv1.Deployment{}
	err := p.Client.Get(ctx, types.NamespacedName{Name: p.name, Namespace: p.namespace}, found)
	if err != nil {
		return false, err
	}
	if found.Status.ReadyReplicas == 0 {
		return true, nil
	}
	return false, nil
}

func (p *pyNode) Start(ctx context.Context) error {
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		dep := &appsv1.Deployment{}
		err := p.Client.Get(ctx, types.NamespacedName{Name: p.name, Namespace: p.namespace}, dep)
		if err != nil {
			return err
		}
		dep.Spec.Replicas = pointer.Int32(1)
		err = p.Client.Update(ctx, dep)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		p.logger.Error(err, "start node failed", "name", p.name, "namespace", p.namespace)
		return err
	}
	p.logger.Info("start node success", "name", p.name, "namespace", p.namespace)
	return nil
}

func (p *pyNode) Stop(ctx context.Context) error {
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		dep := &appsv1.Deployment{}
		err := p.Client.Get(ctx, types.NamespacedName{Name: p.name, Namespace: p.namespace}, dep)
		if err != nil {
			return err
		}
		dep.Spec.Replicas = pointer.Int32(0)
		err = p.Client.Update(ctx, dep)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		p.logger.Error(err, "stop node failed", "name", p.name, "namespace", p.namespace)
		return err
	}
	p.logger.Info("stop node success", "name", p.name, "namespace", p.namespace)
	return nil
}

func newPyNode(namespace, name string, client client.Client, logger logr.Logger) (Node, error) {
	return &pyNode{
		Client:    client,
		namespace: namespace,
		name:      name,
		logger:    logger,
	}, nil
}

func init() {
	Register(citav1.CloudConfig, newCloudConfigNode)
	Register(citav1.PythonOperator, newPyNode)
}
