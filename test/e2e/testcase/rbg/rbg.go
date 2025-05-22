package rbg

import (
	"fmt"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilpointer "k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	lwsv1 "sigs.k8s.io/lws/api/leaderworkerset/v1"
	"sigs.k8s.io/rbgs/api/workloads/v1alpha1"
	"sigs.k8s.io/rbgs/pkg/utils"
	"sigs.k8s.io/rbgs/test/e2e/framework"
)

var (
	baseRbg = v1alpha1.RoleBasedGroup{
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

	lwsRbg = v1alpha1.RoleBasedGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-lws-rbg",
			Namespace: framework.DefaultNamespace,
		},
		Spec: v1alpha1.RoleBasedGroupSpec{
			Roles: []v1alpha1.RoleSpec{
				{
					Name: "prefill",
					Workload: v1alpha1.WorkloadSpec{
						APIVersion: "leaderworkerset.x-k8s.io/v1",
						Kind:       "LeaderWorkerSet",
					},
					Replicas: utilpointer.Int32Ptr(1),
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "prefill",
									Image: framework.DefaultImage,
								},
							},
						},
					},
					LeaderWorkerSet: v1alpha1.LeaderWorkerTemplate{
						Size: utilpointer.Int32Ptr(2),
					},
				},
				{
					Name: "decode",
					Workload: v1alpha1.WorkloadSpec{
						APIVersion: "leaderworkerset.x-k8s.io/v1",
						Kind:       "LeaderWorkerSet",
					},
					Replicas: utilpointer.Int32Ptr(1),
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "decode",
									Image: framework.DefaultImage,
								},
							},
						},
					},
					LeaderWorkerSet: v1alpha1.LeaderWorkerTemplate{
						Size: utilpointer.Int32Ptr(2),
					},
				},
			},
		},
	}
)

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
				gomega.Expect(err).To(gomega.BeNil())

				err = f.ExpectRbgEqual(rbg)
				gomega.Expect(err).To(gomega.BeNil())
			})
		})

		ginkgo.Context("rbg with dependency", func() {
			ginkgo.It("rbg with dependency", func() {
				rbg := baseRbg.DeepCopy()
				role := v1alpha1.RoleSpec{
					Name: "role-2",
					Workload: v1alpha1.WorkloadSpec{
						APIVersion: "apps/v1",
						Kind:       "StatefulSet",
					},
					Dependencies: []string{"role-1"},
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
				rbg.Spec.Roles = append(rbg.Spec.Roles, role)
				err := f.Client.Create(f.Ctx, rbg)
				gomega.Expect(err).To(gomega.BeNil())

				err = f.ExpectRbgEqual(rbg)
				gomega.Expect(err).To(gomega.BeNil())
			})
		})
	})
}

func RunLwsRbgTestCases(f *framework.Framework) {
	ginkgo.Describe("rbg controller", func() {
		ginkgo.Context("one rbg with lws workload", func() {
			ginkgo.It("rbg with lws workload", func() {
				obj := lwsRbg.DeepCopy()
				leader := corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": "prefill-leader",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name: "prefill",
								Env: []corev1.EnvVar{
									{
										Name:  "prefill-role",
										Value: "leader",
									},
								},
							},
						},
					},
				}
				worker := corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": "prefill-worker",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name: "prefill",
								Env: []corev1.EnvVar{
									{
										Name:  "prefill-role",
										Value: "worker",
									},
								},
							},
						},
					},
				}
				obj.Spec.Roles[0].LeaderWorkerSet.PatchLeaderTemplate = runtime.RawExtension{
					Raw: []byte(utils.DumpJSON(leader)),
				}
				obj.Spec.Roles[0].LeaderWorkerSet.PatchWorkerTemplate = runtime.RawExtension{
					Raw: []byte(utils.DumpJSON(worker)),
				}
				leader = corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": "decode-leader",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name: "decode",
								Env: []corev1.EnvVar{
									{
										Name:  "decode-role",
										Value: "leader",
									},
								},
							},
						},
					},
				}
				worker = corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": "decode-worker",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name: "decode",
								Env: []corev1.EnvVar{
									{
										Name:  "decode-role",
										Value: "worker",
									},
								},
							},
						},
					},
				}
				obj.Spec.Roles[1].LeaderWorkerSet.PatchLeaderTemplate = runtime.RawExtension{
					Raw: []byte(utils.DumpJSON(leader)),
				}
				obj.Spec.Roles[1].LeaderWorkerSet.PatchWorkerTemplate = runtime.RawExtension{
					Raw: []byte(utils.DumpJSON(worker)),
				}
				err := f.Client.Create(f.Ctx, obj)
				gomega.Expect(err).To(gomega.BeNil())

				// check rbg ready
				gomega.Eventually(func() bool {
					err = f.Client.Get(f.Ctx, client.ObjectKey{Namespace: obj.Namespace, Name: obj.Name}, obj)
					gomega.Expect(err).To(gomega.BeNil())
					conditionReady := false
					for _, cond := range obj.Status.Conditions {
						if cond.Type == string(v1alpha1.RoleBasedGroupReady) && cond.Status == metav1.ConditionTrue {
							conditionReady = true
						}
					}
					if !conditionReady {
						return false
					}
					for _, role := range obj.Status.RoleStatuses {
						if role.ReadyReplicas != 1 {
							return false
						}
					}
					return true
				}, 30*time.Second, time.Second).Should(gomega.Equal(true))

				time.Sleep(time.Second * 5)

				// scale, and update env
				err = f.Client.Get(f.Ctx, client.ObjectKey{Namespace: obj.Namespace, Name: obj.Name}, obj)
				gomega.Expect(err).To(gomega.BeNil())
				obj.Spec.Roles[0].Replicas = utilpointer.Int32Ptr(2)
				obj.Spec.Roles[1].Replicas = utilpointer.Int32Ptr(2)
				obj.Spec.Roles[1].Template.Spec.Containers[0].Env = []corev1.EnvVar{
					{
						Name:  "decode-version",
						Value: "v2",
					},
				}
				err = f.Client.Update(f.Ctx, obj)
				gomega.Expect(err).To(gomega.BeNil())
				// check rbg ready
				gomega.Eventually(func() bool {
					err = f.Client.Get(f.Ctx, client.ObjectKey{Namespace: obj.Namespace, Name: obj.Name}, obj)
					gomega.Expect(err).To(gomega.BeNil())
					conditionReady := false
					for _, cond := range obj.Status.Conditions {
						if cond.Type == string(v1alpha1.RoleBasedGroupReady) && cond.Status == metav1.ConditionTrue {
							conditionReady = true
						}
					}
					if !conditionReady {
						return false
					}
					for _, role := range obj.Status.RoleStatuses {
						if role.ReadyReplicas != 2 {
							return false
						}
					}
					return true
				}, 60*time.Second, time.Second).Should(gomega.Equal(true))

				decodelws := &lwsv1.LeaderWorkerSet{}
				err = f.Client.Get(f.Ctx, client.ObjectKey{Namespace: obj.Namespace, Name: fmt.Sprintf("%s-decode", obj.Name)}, decodelws)
				gomega.Expect(err).To(gomega.BeNil())
				version := false
				for _, env := range decodelws.Spec.LeaderWorkerTemplate.LeaderTemplate.Spec.Containers[0].Env {
					if env.Name == "decode-version" && env.Value == "v2" {
						version = true
					}
				}
				gomega.Expect(version).To(gomega.Equal(true))
				version = false
				for _, env := range decodelws.Spec.LeaderWorkerTemplate.WorkerTemplate.Spec.Containers[0].Env {
					if env.Name == "decode-version" && env.Value == "v2" {
						version = true
					}
				}
				gomega.Expect(version).To(gomega.Equal(true))
			})
		})
	})
}
