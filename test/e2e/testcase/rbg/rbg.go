package rbg

import (
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/rbgs/api/workloads/v1alpha1"
	"sigs.k8s.io/rbgs/test/e2e/framework"
)

var baseRbg = v1alpha1.RoleBasedGroup{
	ObjectMeta: metav1.ObjectMeta{
		Name:      framework.DefaultRbgName,
		Namespace: framework.DefaultNamespace,
	},
	Spec: v1alpha1.RoleBasedGroupSpec{
		Roles: []v1alpha1.RoleSpec{
			{
				Name: "role-1",
				Workload: v1alpha1.WorkloadSpec{
					APIVersion: "apps/v1",
					Kind:       "StatefulSet",
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-1",
						Namespace: framework.DefaultNamespace,
						Labels: map[string]string{
							"app": "pod-1",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "container-1",
								Image: framework.DefaultImage,
							},
						},
					},
				},
			},
		},
	},
}

func RunRbgControllerTestCases(f *framework.Framework) {
	ginkgo.Describe("rbg controller", func() {
		ginkgo.Context("base rbg", func() {
			ginkgo.It("base rbg", func() {
				rbg := baseRbg.DeepCopy()
				err := f.Client.Create(f.Ctx, rbg)
				gomega.Expect(err).To(gomega.BeNil())
			})
		})

		ginkgo.Context("one rbg with different workloads", func() {
			ginkgo.It("rbg with different workload", func() {
				rbg := baseRbg.DeepCopy()
				deployRole := v1alpha1.RoleSpec{
					Name: "role-2",
					Workload: v1alpha1.WorkloadSpec{
						Kind:       "Deployment",
						APIVersion: "apps/v1",
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "pod-2",
							Namespace: framework.DefaultNamespace,
							Labels: map[string]string{
								"app": "pod-2",
							},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "container-2",
									Image: framework.DefaultImage,
								},
							},
						},
					},
				}
				rbg.Spec.Roles = append(rbg.Spec.Roles, deployRole)
				err := f.Client.Create(f.Ctx, rbg)

				err = f.ExpectRbgEqual(rbg)
				gomega.Expect(err).To(gomega.BeNil())
			})
		})
	})
}
