package citaswitchovercontroller

import (
	"context"
	"errors"
	k8upv1 "github.com/k8up-io/k8up/v2/api/v1"
	citav1 "github.com/k8up-io/k8up/v2/api/v1cita"
	"github.com/k8up-io/k8up/v2/domain"
	"github.com/k8up-io/k8up/v2/operator/cfg"
	"github.com/k8up-io/k8up/v2/operator/executor"
	"github.com/k8up-io/k8up/v2/operator/job"
	batchv1 "k8s.io/api/batch/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type SwitchoverExecutor struct {
	executor.Generic
	switchover *citav1.Switchover
	sourceNode domain.Node
	destNode   domain.Node
}

// NewSwitchoverExecutor will return a new executor for Switchover jobs.
func NewSwitchoverExecutor(ctx context.Context, config job.Config) (*SwitchoverExecutor, error) {

	switchover, ok := config.Obj.(*citav1.Switchover)
	if !ok {
		return nil, errors.New("object is not a switchover")
	}
	// create source node
	sourceNode, err := domain.CreateNode(citav1.CloudConfig, switchover.Namespace, switchover.Spec.SourceNode, config.Client,
		controllerruntime.LoggerFrom(ctx))
	if err != nil {
		return nil, err
	}

	// create dest object
	destNode, err := domain.CreateNode(citav1.CloudConfig, switchover.Namespace, switchover.Spec.DestNode, config.Client,
		controllerruntime.LoggerFrom(ctx))
	if err != nil {
		return nil, err
	}

	return &SwitchoverExecutor{
		Generic:    executor.Generic{Config: config},
		sourceNode: sourceNode,
		destNode:   destNode,
		switchover: switchover}, nil
}

// GetConcurrencyLimit returns the concurrent jobs limit
func (s *SwitchoverExecutor) GetConcurrencyLimit() int {
	// todo: unused
	return cfg.Config.GlobalConcurrentRestoreJobsLimit
}

// Execute creates the actual batch.job on the k8s api.
func (s *SwitchoverExecutor) Execute(ctx context.Context) error {
	log := controllerruntime.LoggerFrom(ctx)

	err := s.createServiceAccountAndBinding(ctx)
	if err != nil {
		return err
	}

	// wait stop source chain node
	allStopped, err := s.StopChainNodes(ctx)
	if err != nil {
		return err
	}
	if !allStopped {
		return nil
	}

	switchoverJob, err := s.createSwitchoverObject(ctx, s.switchover)
	if err != nil {
		log.Error(err, "unable to create or update switchover object")
		s.SetConditionFalseWithMessage(ctx, k8upv1.ConditionReady, k8upv1.ReasonCreationFailed, "unable to create switchover object: %v", err)
		return nil
	}

	s.SetStarted(ctx, "the job '%v/%v' was created", switchoverJob.Namespace, switchoverJob.Name)

	return nil
}

func (s *SwitchoverExecutor) cleanupOldSwitchover(ctx context.Context, switchover *citav1.Switchover) {
	s.CleanupOldResources(ctx, &citav1.SwitchoverList{}, switchover.Namespace, switchover)
}

func (s *SwitchoverExecutor) createSwitchoverObject(ctx context.Context, switchover *citav1.Switchover) (*batchv1.Job, error) {
	batchJob := &batchv1.Job{}
	batchJob.Name = s.jobName()
	batchJob.Namespace = switchover.Namespace
	_, err := controllerutil.CreateOrUpdate(ctx, s.Client, batchJob, func() error {
		mutateErr := job.MutateBatchJob(batchJob, switchover, s.Config)
		if mutateErr != nil {
			return mutateErr
		}

		batchJob.Spec.Template.Spec.ServiceAccountName = cfg.Config.ServiceAccount

		args := s.args()
		batchJob.Spec.Template.Spec.Containers[0].Args = args
		batchJob.Spec.Template.Spec.Containers[0].Command = []string{"/usr/local/bin/k8up", "switchover"}

		return nil
	})
	return batchJob, err
}

func (s *SwitchoverExecutor) jobName() string {
	return citav1.SwitchoverType.String() + "-" + s.Obj.GetName()
}

func (s *SwitchoverExecutor) StartChainNodes(ctx context.Context) {
	log := controllerruntime.LoggerFrom(ctx)

	destNodeStopped, err := s.destNode.CheckStopped(ctx)
	if err != nil {
		log.Error(err, "check chain node stopped failed", "name", s.Obj.GetName(), "namespace", s.Obj.GetNamespace(), "node", s.switchover.Spec.DestNode)
		s.SetConditionFalseWithMessage(ctx, citav1.ConditionCheckChainNodeReady, k8upv1.ReasonFailed, "check chain node stopped failed: %v", err)
	}
	if destNodeStopped {
		err = s.destNode.Start(ctx)
		if err != nil {
			log.Error(err, "start chain node failed", "name", s.Obj.GetName(), "namespace", s.Obj.GetNamespace(), "node", s.switchover.Spec.DestNode)
			s.SetConditionFalseWithMessage(ctx, citav1.ConditionStartChainNodeReady, k8upv1.ReasonFailed, "start chain node failed: %v", err)
			return
		}
	}

	sourceNodeStopped, err := s.sourceNode.CheckStopped(ctx)
	if err != nil {
		log.Error(err, "check chain node stopped failed", "name", s.Obj.GetName(), "namespace", s.Obj.GetNamespace(), "node", s.switchover.Spec.SourceNode)
		s.SetConditionFalseWithMessage(ctx, citav1.ConditionCheckChainNodeReady, k8upv1.ReasonFailed, "check chain node stopped failed: %v", err)
	}
	if sourceNodeStopped {
		err = s.sourceNode.Start(ctx)
		if err != nil {
			log.Error(err, "start chain node failed", "name", s.Obj.GetName(), "namespace", s.Obj.GetNamespace(), "node", s.switchover.Spec.SourceNode)
			s.SetConditionFalseWithMessage(ctx, citav1.ConditionStartChainNodeReady, k8upv1.ReasonFailed, "start chain node failed: %v", err)
			return
		}
	}

	s.SetConditionTrueWithMessage(ctx, citav1.ConditionStartChainNodeReady, k8upv1.ReasonReady, "start all chain node successful")
	return
}

// StopChainNodes stop source node and dest node
func (s *SwitchoverExecutor) StopChainNodes(ctx context.Context) (bool, error) {
	log := controllerruntime.LoggerFrom(ctx)
	// stop source node
	err := s.sourceNode.Stop(ctx)
	if err != nil {
		log.Error(err, "stop chain node failed", "name", s.Obj.GetName(), "namespace", s.Obj.GetNamespace(), "node", s.switchover.Spec.SourceNode)
		s.SetConditionFalseWithMessage(ctx, citav1.ConditionStopChainNodeReady, k8upv1.ReasonFailed, "stop chain node failed: %v", err)
		return false, err
	}
	// stop dest node
	err = s.destNode.Stop(ctx)
	if err != nil {
		log.Error(err, "stop chain node failed", "name", s.Obj.GetName(), "namespace", s.Obj.GetNamespace(), "node", s.switchover.Spec.DestNode)
		s.SetConditionFalseWithMessage(ctx, citav1.ConditionStopChainNodeReady, k8upv1.ReasonFailed, "stop chain node failed: %v", err)
		return false, err
	}
	sourceNodeStopped, err := s.sourceNode.CheckStopped(ctx)
	if err != nil {
		log.Error(err, "check chain node stopped failed", "name", s.Obj.GetName(), "namespace", s.Obj.GetNamespace(), "node", s.switchover.Spec.SourceNode)
		s.SetConditionFalseWithMessage(ctx, citav1.ConditionCheckChainNodeReady, k8upv1.ReasonFailed, "check chain node stopped failed: %v", err)
		return false, err
	}

	destNodeStopped, err := s.destNode.CheckStopped(ctx)
	if err != nil {
		log.Error(err, "check chain node stopped failed", "name", s.Obj.GetName(), "namespace", s.Obj.GetNamespace(), "node", s.switchover.Spec.DestNode)
		s.SetConditionFalseWithMessage(ctx, citav1.ConditionCheckChainNodeReady, k8upv1.ReasonFailed, "check chain node stopped failed: %v", err)
		return false, err
	}

	if !sourceNodeStopped || !destNodeStopped {
		s.SetConditionTrueWithMessage(ctx, citav1.ConditionCheckChainNodeReady, k8upv1.ReasonWaiting, "waiting for chain node stopped")
		return false, nil
	}
	return true, nil
}

func (s *SwitchoverExecutor) args() []string {
	return []string{"--namespace", s.switchover.Namespace,
		"--source-node", s.switchover.Spec.SourceNode,
		"--dest-node", s.switchover.Spec.DestNode}
}
