// executor contains the logic that is needed to apply the actual k8s job objects to a cluster.
// each job type should implement its own executor that handles its own job creation.
// There are various methods that provide default env vars and batch.job scaffolding.

package executor

import (
	"context"
	"k8s.io/apimachinery/pkg/api/errors"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8upv1 "github.com/k8up-io/k8up/v2/api/v1"
	"github.com/k8up-io/k8up/v2/operator/executor/cleaner"
	"github.com/k8up-io/k8up/v2/operator/job"
)

type Generic struct {
	job.Config
}

// listOldResources retrieves a list of the given resource type in the given namespace and fills the Item property
// of objList. On errors, the error is being logged and the Scrubbed condition set to False with reason RetrievalFailed.
func (g *Generic) listOldResources(ctx context.Context, namespace string, objList client.ObjectList) error {
	log := controllerruntime.LoggerFrom(ctx)
	err := g.Client.List(ctx, objList, &client.ListOptions{
		Namespace: namespace,
	})
	if err != nil {
		log.Error(err, "could not list objects to cleanup old resources")
		g.SetConditionFalseWithMessage(ctx, k8upv1.ConditionScrubbed, k8upv1.ReasonRetrievalFailed, "could not list objects to cleanup old resources: %v", err)
		return err
	}
	return nil
}

type jobObjectList interface {
	client.ObjectList

	GetJobObjects() k8upv1.JobObjectList
}

func (g *Generic) CleanupOldResources(ctx context.Context, typ jobObjectList, namespace string, limits cleaner.GetJobsHistoryLimiter) {
	err := g.listOldResources(ctx, namespace, typ)
	if err != nil {
		return
	}

	cl := cleaner.NewObjectCleaner(g.Client, limits)
	deleted, err := cl.CleanOldObjects(ctx, typ.GetJobObjects())
	if err != nil {
		g.SetConditionFalseWithMessage(ctx, k8upv1.ConditionScrubbed, k8upv1.ReasonDeletionFailed, "could not cleanup old resources: %s", err.Error())
		return
	}
	g.SetConditionTrueWithMessage(ctx, k8upv1.ConditionScrubbed, k8upv1.ReasonSucceeded, "Deleted %d resources", deleted)

}

// CreateObjectIfNotExisting tries to create the given object, but ignores AlreadyExistsError.
// If it fails for any other reason, the Ready condition is set to False with the error message and reason.
func (g *Generic) CreateObjectIfNotExisting(ctx context.Context, obj client.Object) error {
	err := g.Client.Create(ctx, obj)
	if err != nil && !errors.IsAlreadyExists(err) {
		g.SetConditionFalseWithMessage(
			ctx,
			k8upv1.ConditionReady,
			k8upv1.ReasonCreationFailed,
			"unable to create %v '%v/%v': %v",
			obj.GetObjectKind().GroupVersionKind().Kind,
			obj.GetNamespace(), obj.GetName(), err.Error())
		return err
	}
	return nil
}

func BuildIncludePathArgs(includePathList []string) []string {
	var args []string
	for i := range includePathList {
		args = append(args, "--path", includePathList[i])
	}
	return args
}

//func (g *Generic) getObject(namespace, name string, object client.Object) error {
//	err := g.Client.Get(g.CTX, types.NamespacedName{Namespace: namespace, Name: name}, object)
//	if err != nil {
//		return err
//	}
//	return nil
//}
