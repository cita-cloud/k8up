package citabase

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"

	k8upv1 "github.com/k8up-io/k8up/v2/api/v1"
	citav1 "github.com/k8up-io/k8up/v2/api/v1cita"
	"github.com/k8up-io/k8up/v2/domain"
	"github.com/k8up-io/k8up/v2/operator/executor"
	"github.com/k8up-io/k8up/v2/operator/job"
)

type CITABase struct {
	executor.Generic
	node domain.Node
}

func NewCITABase(config job.Config, node domain.Node) CITABase {
	return CITABase{
		Generic: executor.Generic{Config: config},
		node:    node,
	}
}

func (c *CITABase) StopChainNode(ctx context.Context) (bool, error) {
	log := controllerruntime.LoggerFrom(ctx)
	stopped, err := c.node.CheckStopped(ctx)
	if err != nil {
		log.Error(err, "check chain node stopped failed", "name", c.Obj.GetName(), "namespace", c.Obj.GetNamespace())
		c.SetConditionFalseWithMessage(ctx, citav1.ConditionStopChainNodeReady, k8upv1.ReasonFailed, "check chain node stopped failed: %v", err)
		return false, err
	}
	if !stopped {
		// stop chain node
		err := c.node.Stop(ctx)
		if err != nil {
			log.Error(err, "stop chain node failed", "name", c.Obj.GetName(), "namespace", c.Obj.GetNamespace())
			c.SetConditionFalseWithMessage(ctx, citav1.ConditionStopChainNodeReady, k8upv1.ReasonFailed, "stop chain node failed: %v", err)
			return false, err
		}

		c.SetConditionUnknownWithMessage(ctx, citav1.ConditionStopChainNodeReady, k8upv1.ReasonWaiting, "waiting for chain node stopped")
		return false, nil
	}
	c.SetConditionTrueWithMessage(ctx, citav1.ConditionStopChainNodeReady, k8upv1.ReasonReady,
		"the cita node %s/%s has benn stopped",
		c.Obj.GetName(), c.Obj.GetNamespace())
	return true, nil
}

func (c *CITABase) StartChainNode(ctx context.Context) {
	log := controllerruntime.LoggerFrom(ctx)
	// start chain node
	err := c.node.Start(ctx)
	if err != nil {
		log.Error(err, "start chain node failed", "name", c.Obj.GetName(), "namespace", c.Obj.GetNamespace())
		c.SetConditionFalseWithMessage(ctx, citav1.ConditionStartChainNodeReady, k8upv1.ReasonFailed, "start chain node failed: %v", err)
		return
	}
	c.SetConditionTrueWithMessage(ctx, citav1.ConditionStartChainNodeReady, k8upv1.ReasonReady,
		"the cita node %s/%s has benn started",
		c.Obj.GetName(), c.Obj.GetNamespace())
	return
}

func (c *CITABase) GetMountPoint(path string) (string, error) {
	res := strings.Split(path, "/")
	if len(res) < 2 {
		return "", fmt.Errorf("path invaild")
	}
	return fmt.Sprintf("/%s", res[1]), nil
}

func (c *CITABase) GetVolume(ctx context.Context) (corev1.Volume, error) {
	return c.node.GetVolume(ctx)
}

func (c *CITABase) GetPVCInfo(ctx context.Context) (corev1.ResourceRequirements, error) {
	return c.node.GetPVCInfo(ctx)
}

func (c *CITABase) GetCryptoAndConsensus(ctx context.Context) (string, string, error) {
	return c.node.GetCryptoAndConsensus(ctx)
}
