package citabackupcontroller

import (
	"context"
	"fmt"
	"github.com/k8up-io/k8up/v2/operator/executor"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"

	"github.com/k8up-io/k8up/v2/operator/cfg"
)

func (b *BackupExecutor) createServiceAccountAndBinding(ctx context.Context) error {
	sa := &corev1.ServiceAccount{}
	sa.Name = cfg.Config.ServiceAccount
	sa.Namespace = b.backup.Namespace
	_, err := controllerruntime.CreateOrUpdate(ctx, b.Config.Client, sa, func() error {
		return nil
	})
	if err != nil {
		return err
	}

	role := &rbacv1.Role{}
	role.Name = cfg.Config.PodExecRoleName
	role.Namespace = b.backup.Namespace
	_, err = controllerruntime.CreateOrUpdate(ctx, b.Config.Client, role, func() error {
		role.Rules = []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"pods", "pods/exec"},
				Verbs:     []string{"*"},
			},
			{
				Verbs: []string{
					"get",
					"list",
					"watch",
					"update",
				},
				APIGroups: []string{
					"apps",
				},
				Resources: []string{
					"statefulsets",
				},
			},
		}
		return nil
	})

	if err != nil {
		return err
	}
	roleBinding := &rbacv1.RoleBinding{}
	roleBinding.Name = cfg.Config.PodExecRoleName + "-namespaced"
	roleBinding.Namespace = b.backup.Namespace
	_, err = controllerruntime.CreateOrUpdate(ctx, b.Config.Client, roleBinding, func() error {
		roleBinding.Subjects = []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Namespace: b.backup.Namespace,
				Name:      sa.Name,
			},
		}
		roleBinding.RoleRef = rbacv1.RoleRef{
			Kind:     "Role",
			Name:     role.Name,
			APIGroup: "rbac.authorization.k8s.io",
		}
		return nil
	})
	return err
}

func (b *BackupExecutor) setupEnvVars() ([]corev1.EnvVar, error) {
	vars := executor.NewEnvVarConverter()

	if b.backup != nil {
		if b.backup.Spec.Backend != nil {
			for key, value := range b.backup.Spec.Backend.GetCredentialEnv() {
				vars.SetEnvVarSource(key, value)
			}
			vars.SetString(cfg.ResticRepositoryEnvName, b.backup.Spec.Backend.String())
		}
	}

	vars.SetStringOrDefault("STATS_URL", b.backup.Spec.StatsURL, cfg.Config.GlobalStatsURL)
	vars.SetStringOrDefault("PROM_URL", b.backup.Spec.PromURL, cfg.Config.PromURL)
	vars.SetString("BACKUPCOMMAND_ANNOTATION", cfg.Config.BackupCommandAnnotation)
	vars.SetString("FILEEXTENSION_ANNOTATION", cfg.Config.FileExtensionAnnotation)

	err := vars.Merge(executor.DefaultEnv(b.backup.GetNamespace()))
	if err != nil {
		return nil, fmt.Errorf("cannot merge environment variables: %w", err)
	}
	return vars.Convert(), nil
}
