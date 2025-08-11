package framework

import (
	"fmt"

	"github.com/onsi/gomega"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/rbgs/api/workloads/v1alpha1"
	"sigs.k8s.io/rbgs/pkg/scale"
	"sigs.k8s.io/rbgs/test/utils"
)

func (f *Framework) ExpectRbgEqual(rbg *v1alpha1.RoleBasedGroup) {
	logger := log.FromContext(f.Ctx).WithValues("rbg", rbg.Name)
	newRbg := &v1alpha1.RoleBasedGroup{}
	gomega.Eventually(func() bool {
		err := f.Client.Get(f.Ctx, client.ObjectKey{
			Name:      rbg.Name,
			Namespace: rbg.Namespace,
		}, newRbg)
		if err != nil {
			if !apierrors.IsNotFound(err) {
				logger.Error(err, "get rbg error")
			}
			return false
		}
		return true
	}, utils.Timeout, utils.Interval).Should(gomega.BeTrue())

	// check workload exists
	for _, role := range rbg.Spec.Roles {
		wlCheck, err := NewWorkloadEqualChecker(f.Ctx, f.Client, role.Workload.String())
		gomega.Expect(err).To(gomega.BeNil())

		gomega.Eventually(func() bool {
			err := wlCheck.ExpectWorkloadEqual(rbg, role)
			if err != nil {
				logger.V(1).Info("workload not equal, wait next time", "reason", err.Error())
			}
			return err == nil
		}, utils.Timeout, utils.Interval).Should(gomega.BeTrue())
	}
}

func (f *Framework) ExpectRbgDeleted(rbg *v1alpha1.RoleBasedGroup) {
	newRbg := &v1alpha1.RoleBasedGroup{}
	gomega.Eventually(func() bool {
		err := f.Client.Get(f.Ctx, client.ObjectKey{
			Name:      rbg.Name,
			Namespace: rbg.Namespace,
		}, newRbg)
		if apierrors.IsNotFound(err) {
			return true
		}

		return false
	}, utils.Timeout, utils.Interval).Should(gomega.BeTrue())
}

func (f *Framework) ExpectRbgCondition(rbg *v1alpha1.RoleBasedGroup,
	conditionType v1alpha1.RoleBasedGroupConditionType, conditionStatus metav1.ConditionStatus) bool {
	logger := log.FromContext(f.Ctx)

	newRbg := &v1alpha1.RoleBasedGroup{}
	gomega.Eventually(func() bool {
		err := f.Client.Get(f.Ctx, client.ObjectKey{
			Name:      rbg.Name,
			Namespace: rbg.Namespace,
		}, newRbg)
		if err != nil {
			logger.V(1).Error(err, "get rbg error")
		}
		return err == nil
	}, utils.Timeout, utils.Interval).Should(gomega.BeTrue())

	for _, condition := range newRbg.Status.Conditions {
		if condition.Type == string(conditionType) && condition.Status == conditionStatus {
			return true
		}

	}
	return false
}

func (f *Framework) ExpectWorkloadLabelContains(rbg *v1alpha1.RoleBasedGroup, role v1alpha1.RoleSpec, labels ...map[string]string) {
	wlCheck, err := NewWorkloadEqualChecker(f.Ctx, f.Client, role.Workload.String())
	gomega.Expect(err).To(gomega.BeNil())

	gomega.Eventually(func() bool {
		return wlCheck.ExpectLabelContains(rbg, role, labels...) == nil
	}, utils.Timeout, utils.Interval).Should(gomega.BeTrue())
}

func (f *Framework) ExpectWorkloadNotExist(rbg *v1alpha1.RoleBasedGroup, role v1alpha1.RoleSpec) {
	wlCheck, err := NewWorkloadEqualChecker(f.Ctx, f.Client, role.Workload.String())
	gomega.Expect(err).To(gomega.BeNil())

	gomega.Eventually(func() bool {
		return wlCheck.ExpectWorkloadNotExist(rbg, role) == nil
	}, utils.Timeout, utils.Interval).Should(gomega.BeTrue())
}

func (f *Framework) ExpectRoleScalingAdapterEqual(rbg *v1alpha1.RoleBasedGroup, role v1alpha1.RoleSpec, expectedReplicas *int32) {
	logger := log.FromContext(f.Ctx).WithValues("rbg", rbg.Name)

	gomega.Eventually(func() bool {
		rbgSa := &v1alpha1.RoleBasedGroupScalingAdapter{}
		err := f.Client.Get(f.Ctx, client.ObjectKey{
			Name:      scale.GenerateScalingAdapterName(rbg.Name, role.Name),
			Namespace: rbg.Namespace,
		}, rbgSa)
		if err != nil {
			if !apierrors.IsNotFound(err) {
				logger.Error(err, "get rbg error")
			}
			return false
		}

		scale := &autoscalingv1.Scale{}
		if err := f.Client.SubResource("scale").Get(f.Ctx, rbgSa, scale); err != nil {
			logger.Info("get subresource-scale for rbg scaling adapter failed, wait next time", "reason", err.Error())
			return false
		}
		if err := f.Client.Get(f.Ctx, client.ObjectKey{
			Name:      rbg.Name,
			Namespace: rbg.Namespace,
		}, rbg); err != nil {
			logger.Info("get subresource-scale for rbg scaling adapter failed, wait next time", "reason", err.Error())
			return false
		}
		newRoleFound := false
		for _, newRole := range rbg.Spec.Roles {
			if newRole.Name == role.Name {
				role, newRoleFound = newRole, true
				break
			}
		}
		if !newRoleFound {
			return false
		}
		err = expectRbgScalingAdapterEqual(rbgSa, rbg, role, scale, expectedReplicas)
		if err != nil {
			logger.Info("rbgScalingAdapter not equal, wait next time", "reason", err.Error())
		}
		return err == nil
	}, utils.Timeout, utils.Interval).Should(gomega.BeTrue())
}

func (f *Framework) ExpectRbgScalingAdapterEqual(rbg *v1alpha1.RoleBasedGroup) {
	for _, role := range rbg.Spec.Roles {
		if role.ScalingAdapter == nil || !role.ScalingAdapter.Enable {
			f.ExpectScalingAdapterNotExist(rbg, role)
		} else {
			f.ExpectRoleScalingAdapterEqual(rbg, role, nil)
		}
	}
}
func (f *Framework) ExpectScalingAdapterNotExist(rbg *v1alpha1.RoleBasedGroup, role v1alpha1.RoleSpec) {
	checkScalingAdapterNotExist := func() error {
		rbgSa := &v1alpha1.RoleBasedGroupScalingAdapter{}
		err := f.Client.Get(f.Ctx, client.ObjectKey{
			Name:      scale.GenerateScalingAdapterName(rbg.Name, role.Name),
			Namespace: rbg.Namespace,
		}, rbgSa)
		if err == nil {
			return fmt.Errorf("rbg scalingAdapter still exists")
		}
		if !apierrors.IsNotFound(err) {
			return err
		}
		return nil
	}

	gomega.Eventually(func() bool {
		return checkScalingAdapterNotExist() == nil
	}, utils.Timeout, utils.Interval).Should(gomega.BeTrue())
}
