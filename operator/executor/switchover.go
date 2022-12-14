package executor

import (
	stderrors "errors"
	citav1 "github.com/k8up-io/k8up/v2/api/v1cita"
	"github.com/k8up-io/k8up/v2/operator/cfg"
	"github.com/k8up-io/k8up/v2/operator/job"
	"github.com/k8up-io/k8up/v2/operator/observer"
	batchv1 "k8s.io/api/batch/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type SwitchoverExecutor struct {
	generic
	switchover *citav1.Switchover
	sourceNode Node
	destNode   Node
}

// NewSwitchoverExecutor returns a new SwitchoverExecutor.
func NewSwitchoverExecutor(config job.Config) *SwitchoverExecutor {
	return &SwitchoverExecutor{
		generic: generic{config},
	}
}

// GetConcurrencyLimit returns the concurrent jobs limit
func (s *SwitchoverExecutor) GetConcurrencyLimit() int {
	return cfg.Config.GlobalConcurrentRestoreJobsLimit
}

// Execute triggers the actual batch.job creation on the cluster.
// It will also register a callback function on the observer so the PreBackupPods can be removed after the backup has finished.
func (s *SwitchoverExecutor) Execute() error {
	switchoverObject, ok := s.Obj.(*citav1.Switchover)
	if !ok {
		return stderrors.New("object is not a block height fallback")
	}
	s.switchover = switchoverObject

	if s.Obj.GetStatus().Started {
		s.RegisterJobSucceededConditionCallback() // ensure that completed jobs can complete backups between operator restarts.
		return nil
	}

	var err error
	// create source node object
	s.sourceNode, err = CreateNode(citav1.CloudConfig, switchoverObject.Namespace, switchoverObject.Spec.SourceNode, s.Client)
	if err != nil {
		return err
	}
	// create dest node object
	s.destNode, err = CreateNode(citav1.CloudConfig, switchoverObject.Namespace, switchoverObject.Spec.DestNode, s.Client)
	if err != nil {
		return err
	}

	err = s.createServiceAccountAndBinding()
	if err != nil {
		return err
	}

	genericJob, err := job.GenerateGenericJob(s.Obj, s.Config)
	if err != nil {
		return err
	}

	return s.startSwitchover(genericJob)
}

func (s *SwitchoverExecutor) createServiceAccountAndBinding() error {
	role, sa, binding := newServiceAccountDefinition(s.switchover.Namespace)
	for _, obj := range []client.Object{&role, &sa, &binding} {
		if err := s.CreateObjectIfNotExisting(obj); err != nil {
			return err
		}
	}
	return nil
}

func (s *SwitchoverExecutor) startSwitchover(job *batchv1.Job) error {
	// stop source node
	err := s.sourceNode.Stop(s.CTX)
	if err != nil {
		return err
	}
	sourceNodeStopped, err := s.sourceNode.CheckStopped(s.CTX)
	if err != nil {
		return err
	}
	// stop dest node
	err = s.destNode.Stop(s.CTX)
	if err != nil {
		return err
	}
	destNodeStopped, err := s.destNode.CheckStopped(s.CTX)
	if err != nil {
		return err
	}
	if !sourceNodeStopped || !destNodeStopped {
		return nil
	}

	s.registerCITANodeCallback()
	s.RegisterJobSucceededConditionCallback()

	job.Spec.Template.Spec.ServiceAccountName = cfg.Config.ServiceAccount

	args := s.args()
	job.Spec.Template.Spec.Containers[0].Args = args
	job.Spec.Template.Spec.Containers[0].Command = []string{"/usr/local/bin/k8up", "switchover"}

	if err = s.CreateObjectIfNotExisting(job); err == nil {
		s.SetStarted("the job '%v/%v' was created", job.Namespace, job.Name)
	}
	return err
}

func (s *SwitchoverExecutor) args() []string {
	return []string{"--namespace", s.switchover.Namespace,
		"--source-node", s.switchover.Spec.SourceNode,
		"--dest-node", s.switchover.Spec.DestNode}
}

func (s *SwitchoverExecutor) registerCITANodeCallback() {
	name := s.GetJobNamespacedName()
	observer.GetObserver().RegisterCallback(name.String(), func(_ observer.ObservableJob) {
		s.startCITANode()
	})
}

func (s *SwitchoverExecutor) startCITANode() {
	err := s.destNode.Start(s.CTX)
	if err != nil {
		// todo event
		return
	}
	// todo event
	err = s.sourceNode.Start(s.CTX)
	if err != nil {
		// todo event
		return
	}
	return
}
