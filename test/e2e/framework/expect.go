package framework

import (
	"context"
	"github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/rbgs/api/workloads/v1alpha1"
	"time"
)

func (f *Framework) ExpectRbgEqual(rbg *v1alpha1.RoleBasedGroup) error {

	newRbg := &v1alpha1.RoleBasedGroup{}
	err := wait.PollUntilContextTimeout(f.Ctx, 3*time.Second, 1*time.Minute, false, func(ctx context.Context) (done bool, err error) {
		err = f.Client.Get(f.Ctx, client.ObjectKey{
			Name:      rbg.Name,
			Namespace: rbg.Namespace,
		}, newRbg)
		if len(newRbg.Status.RoleStatuses) == 0 {
			return false, nil
		}
		if len(newRbg.Status.RoleStatuses) != len(newRbg.Spec.Roles) {
			klog.Infof("role status len %d is not equal with role len %d, wait next time",
				len(newRbg.Status.RoleStatuses), len(newRbg.Spec.Roles))
			return false, nil
		}

		return true, nil
	})

	gomega.Expect(err).To(gomega.BeNil())

	// check workload exists
	for _, role := range rbg.Spec.Roles {
		wlCheck, err := NewWorkloadEqualChecker(f.Ctx, f.Client, role.Workload.APIVersion, role.Workload.Kind)
		if err != nil {
			return err
		}
		if err := wlCheck.ExpectWorkloadEqual(rbg, role); err != nil {
			return err
		}
	}
	return nil
}
