package reconciler

import (
	"context"
	"reflect"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	workloadsv1alpha1 "sigs.k8s.io/rbgs/api/workloads/v1alpha1"
)

func TestStatefulSetReconciler_ConstructRoleStatus(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme) // 添加 StatefulSet 类型支持
	_ = workloadsv1alpha1.AddToScheme(scheme)
	type fields struct {
		client client.Client
		scheme *runtime.Scheme
	}
	type args struct {
		ctx  context.Context
		rbg  *workloadsv1alpha1.RoleBasedGroup
		role *workloadsv1alpha1.RoleSpec
	}

	// 测试用 StatefulSet 模板
	testSTS := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-rbg-test-role", // 假设 GetWorkloadName 生成此名称
			Namespace: "default",
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: ptr.To[int32](3),
		},
		Status: appsv1.StatefulSetStatus{
			ReadyReplicas: 2,
		},
	}

	tests := []struct {
		name             string
		fields           fields
		args             args
		wantStatus       workloadsv1alpha1.RoleStatus
		wantUpdateStatus bool
		wantErr          bool
	}{
		// 用例1: 状态变化需要更新
		{
			name: "status-changed-needs-update",
			fields: fields{
				scheme: scheme,
				client: fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(testSTS.DeepCopy()).
					Build(),
			},
			args: args{
				ctx: context.Background(),
				rbg: &workloadsv1alpha1.RoleBasedGroup{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-rbg",
						Namespace: "default",
					},
					Status: workloadsv1alpha1.RoleBasedGroupStatus{
						RoleStatuses: []workloadsv1alpha1.RoleStatus{
							{
								Name:          "test-role",
								Replicas:      2, // 旧值
								ReadyReplicas: 1, // 旧值
							},
						},
					},
				},
				role: &workloadsv1alpha1.RoleSpec{Name: "test-role"},
			},
			wantStatus: workloadsv1alpha1.RoleStatus{
				Name:          "test-role",
				Replicas:      3,
				ReadyReplicas: 2,
			},
			wantUpdateStatus: true,
			wantErr:          false,
		},
		// 用例2: 状态未变化无需更新
		{
			name: "status-unchanged-no-update",
			fields: fields{
				scheme: scheme,
				client: fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(testSTS.DeepCopy()).
					Build(),
			},
			args: args{
				ctx: context.Background(),
				rbg: &workloadsv1alpha1.RoleBasedGroup{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-rbg",
						Namespace: "default",
					},
					Status: workloadsv1alpha1.RoleBasedGroupStatus{
						RoleStatuses: []workloadsv1alpha1.RoleStatus{
							{
								Name:          "test-role",
								Replicas:      3, // 当前值
								ReadyReplicas: 2, // 当前值
							},
						},
					},
				},
				role: &workloadsv1alpha1.RoleSpec{Name: "test-role"},
			},
			wantStatus: workloadsv1alpha1.RoleStatus{
				Name:          "test-role",
				Replicas:      3,
				ReadyReplicas: 2,
			},
			wantUpdateStatus: false,
			wantErr:          false,
		},
		// 用例3: 首次创建角色状态
		{
			name: "initial-status-creation",
			fields: fields{
				scheme: scheme,
				client: fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(testSTS.DeepCopy()).
					Build(),
			},
			args: args{
				ctx: context.Background(),
				rbg: &workloadsv1alpha1.RoleBasedGroup{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-rbg",
						Namespace: "default",
					},
					Status: workloadsv1alpha1.RoleBasedGroupStatus{
						RoleStatuses: []workloadsv1alpha1.RoleStatus{}, // 无现有状态
					},
				},
				role: &workloadsv1alpha1.RoleSpec{Name: "test-role"},
			},
			wantStatus: workloadsv1alpha1.RoleStatus{
				Name:          "test-role",
				Replicas:      3,
				ReadyReplicas: 2,
			},
			wantUpdateStatus: true,
			wantErr:          false,
		},
		// 用例4: StatefulSet 不存在
		{
			name: "statefulset-not-found",
			fields: fields{
				scheme: scheme,
				client: fake.NewClientBuilder().
					WithScheme(scheme).
					Build(), // 无 StatefulSet
			},
			args: args{
				ctx: context.Background(),
				rbg: &workloadsv1alpha1.RoleBasedGroup{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-rbg",
						Namespace: "default",
					},
					Status: workloadsv1alpha1.RoleBasedGroupStatus{
						RoleStatuses: []workloadsv1alpha1.RoleStatus{
							{Name: "test-role", Replicas: 2, ReadyReplicas: 1},
						},
					},
				},
				role: &workloadsv1alpha1.RoleSpec{Name: "test-role"},
			},
			wantStatus:       workloadsv1alpha1.RoleStatus{},
			wantUpdateStatus: false,
			wantErr:          true, // 预期返回 NotFound 错误
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &StatefulSetReconciler{
				scheme: tt.fields.scheme,
				client: tt.fields.client,
			}
			gotStatus, gotUpdateStatus, err := r.ConstructRoleStatus(tt.args.ctx, tt.args.rbg, tt.args.role)

			// 错误检查
			if (err != nil) != tt.wantErr {
				t.Errorf("testCase %s: error = %v, wantErr %v", tt.name, err, tt.wantErr)
				return
			}

			// 状态对比
			if !reflect.DeepEqual(gotStatus, tt.wantStatus) {
				t.Errorf("testCase %s: gotStatus = %v, want %v", tt.name, gotStatus, tt.wantStatus)
			}

			// 更新标志检查
			if gotUpdateStatus != tt.wantUpdateStatus {
				t.Errorf("testCase %s: gotUpdateStatus = %v, want %v", tt.name, gotUpdateStatus, tt.wantUpdateStatus)
			}
		})
	}
}
