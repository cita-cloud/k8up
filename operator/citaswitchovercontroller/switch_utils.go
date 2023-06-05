package citaswitchovercontroller

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"

	"github.com/k8up-io/k8up/v2/operator/cfg"
)

func (s *SwitchoverExecutor) createServiceAccountAndBinding(ctx context.Context) error {
	sa := &corev1.ServiceAccount{}
	sa.Name = cfg.Config.ServiceAccount
	sa.Namespace = s.switchover.Namespace
	_, err := controllerruntime.CreateOrUpdate(ctx, s.Config.Client, sa, func() error {
		return nil
	})
	if err != nil {
		return err
	}

	role := &rbacv1.Role{}
	role.Name = cfg.Config.PodExecRoleName
	role.Namespace = s.switchover.Namespace
	_, err = controllerruntime.CreateOrUpdate(ctx, s.Config.Client, role, func() error {
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
	roleBinding.Namespace = s.switchover.Namespace
	_, err = controllerruntime.CreateOrUpdate(ctx, s.Config.Client, roleBinding, func() error {
		roleBinding.Subjects = []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Namespace: s.switchover.Namespace,
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
