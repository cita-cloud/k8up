package executor

import (
	"errors"
	"strconv"
	"strings"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	k8upv1 "github.com/k8up-io/k8up/v2/api/v1"
	citav1 "github.com/k8up-io/k8up/v2/api/v1cita"
	"github.com/k8up-io/k8up/v2/operator/cfg"
	"github.com/k8up-io/k8up/v2/operator/job"
	"github.com/k8up-io/k8up/v2/operator/observer"
)

// CITAPruneExecutor will execute the batch.job for Prunes.
type CITAPruneExecutor struct {
	generic
}

// NewCITAPruneExecutor will return a new executor for Prune jobs.
func NewCITAPruneExecutor(config job.Config) *CITAPruneExecutor {
	return &CITAPruneExecutor{
		generic: generic{config},
	}
}

// GetConcurrencyLimit returns the concurrent jobs limit
func (p *CITAPruneExecutor) GetConcurrencyLimit() int {
	return cfg.Config.GlobalConcurrentPruneJobsLimit
}

// Execute creates the actual batch.job on the k8s api.
func (p *CITAPruneExecutor) Execute() error {
	prune, ok := p.Obj.(*citav1.Prune)
	if !ok {
		return errors.New("object is not a prune")
	}

	if prune.GetStatus().Started {
		p.RegisterJobSucceededConditionCallback() // ensure that completed jobs can complete backups between operator restarts.
		return nil
	}

	jobObj, err := job.GenerateGenericJob(prune, p.Config)
	if err != nil {
		p.SetConditionFalseWithMessage(k8upv1.ConditionReady, k8upv1.ReasonCreationFailed, "could not get job template: %v", err)
		return err
	}
	jobObj.GetLabels()[job.K8upExclusive] = "true"

	p.startPrune(jobObj, prune)

	return nil
}

// Exclusive should return true for jobs that can't run while other jobs run.
func (p *CITAPruneExecutor) Exclusive() bool {
	return true
}

func (p *CITAPruneExecutor) startPrune(pruneJob *batchv1.Job, prune *citav1.Prune) {
	p.registerPruneCallback(prune)
	p.RegisterJobSucceededConditionCallback()

	pruneJob.Spec.Template.Spec.Containers[0].Env = p.setupEnvVars(prune)
	prune.Spec.AppendEnvFromToContainer(&pruneJob.Spec.Template.Spec.Containers[0])
	pruneJob.Spec.Template.Spec.Containers[0].Args = append([]string{"-prune"}, BuildTagArgs(prune.Spec.Retention.Tags)...)

	if err := p.Client.Create(p.CTX, pruneJob); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			p.Log.Error(err, "could not create job")
			p.SetConditionFalseWithMessage(k8upv1.ConditionReady, k8upv1.ReasonCreationFailed, "could not create job: %v", err)
			return
		}
	}

	p.SetStarted("the job '%v/%v' was created", pruneJob.Namespace, pruneJob.Name)
}

func (p *CITAPruneExecutor) registerPruneCallback(prune *citav1.Prune) {
	name := p.GetJobNamespacedName()
	observer.GetObserver().RegisterCallback(name.String(), func(_ observer.ObservableJob) {
		p.cleanupOldPrunes(name, prune)
	})
}

func (p *CITAPruneExecutor) cleanupOldPrunes(name types.NamespacedName, prune *citav1.Prune) {
	p.cleanupOldResources(&citav1.PruneList{}, name, prune)
}

func (p *CITAPruneExecutor) setupEnvVars(prune *citav1.Prune) []corev1.EnvVar {
	vars := NewEnvVarConverter()

	// FIXME(mw): this is ugly

	if prune.Spec.Retention.KeepLast > 0 {
		vars.SetString("KEEP_LAST", strconv.Itoa(prune.Spec.Retention.KeepLast))
	}

	if prune.Spec.Retention.KeepHourly > 0 {
		vars.SetString("KEEP_HOURLY", strconv.Itoa(prune.Spec.Retention.KeepHourly))
	}

	if prune.Spec.Retention.KeepDaily > 0 {
		vars.SetString("KEEP_DAILY", strconv.Itoa(prune.Spec.Retention.KeepDaily))
	} else {
		vars.SetString("KEEP_DAILY", "14")
	}

	if prune.Spec.Retention.KeepWeekly > 0 {
		vars.SetString("KEEP_WEEKLY", strconv.Itoa(prune.Spec.Retention.KeepWeekly))
	}

	if prune.Spec.Retention.KeepMonthly > 0 {
		vars.SetString("KEEP_MONTHLY", strconv.Itoa(prune.Spec.Retention.KeepMonthly))
	}

	if prune.Spec.Retention.KeepYearly > 0 {
		vars.SetString("KEEP_YEARLY", strconv.Itoa(prune.Spec.Retention.KeepYearly))
	}

	if len(prune.Spec.Retention.KeepTags) > 0 {
		vars.SetString("KEEP_TAGS", strings.Join(prune.Spec.Retention.KeepTags, ","))
	}

	if prune.Spec.Backend != nil {
		for key, value := range prune.Spec.Backend.GetCredentialEnv() {
			vars.SetEnvVarSource(key, value)
		}
		vars.SetString(cfg.ResticRepositoryEnvName, prune.Spec.Backend.String())
	}

	vars.SetString("PROM_URL", cfg.Config.PromURL)

	err := vars.Merge(DefaultEnv(p.Obj.GetMetaObject().GetNamespace()))
	if err != nil {
		p.Log.Error(err, "error while merging the environment variables", "name", p.Obj.GetMetaObject().GetName(), "namespace", p.Obj.GetMetaObject().GetNamespace())
	}

	return vars.Convert()
}
