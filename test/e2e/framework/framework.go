package framework

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/onsi/ginkgo/v2"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	lwsv1 "sigs.k8s.io/lws/api/leaderworkerset/v1"
	workloadsv1alpha1 "sigs.k8s.io/rbgs/api/workloads/v1alpha1"
)

const DefaultNamespace = "e2e-test"
const DefaultRbgName = "rbg-test"
const DefaultImage = "registry.cn-hangzhou.aliyuncs.com/acs-sample/nginx:latest"

type Framework struct {
	Ctx    context.Context
	Client client.Client
	Logger log.Logger
}

func NewFramework() *Framework {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(workloadsv1alpha1.AddToScheme(scheme))
	utilruntime.Must(lwsv1.AddToScheme(scheme))

	cfg := config.GetConfigOrDie()
	runtimeClient, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {

		panic(fmt.Sprintf("new runtime client error: %s", err.Error()))
	}
	return &Framework{
		Ctx:    context.TODO(),
		Client: runtimeClient,
	}
}

func (f *Framework) BeforeAll() error {
	klog.Info("creating e2e namespace")
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: DefaultNamespace,
		},
	}
	err := wait.PollUntilContextTimeout(f.Ctx, 5*time.Second, 1*time.Minute, true, func(ctx context.Context) (done bool, err error) {
		err = f.Client.Create(f.Ctx, ns)
		if err == nil {
			return true, nil
		}

		if apierrors.IsAlreadyExists(err) {
			delErr := f.Client.Delete(f.Ctx, ns)
			if delErr != nil {
				return false, err
			}
			return false, nil
		}
		return false, err
	})

	if err != nil {
		return err
	}

	return nil
}

func (f *Framework) AfterAll() {
	ginkgo.By("removing e2e namespace")
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: DefaultNamespace,
		},
	}
	_ = f.Client.Delete(f.Ctx, ns)
}

func (f *Framework) AfterEach() {
	rbgList := &workloadsv1alpha1.RoleBasedGroupList{}
	err := f.Client.List(f.Ctx, rbgList)
	if err != nil {
		panic(fmt.Sprintf("failed to list rbg: %s", err.Error()))
	}
	for i := range rbgList.Items {
		rbg := &rbgList.Items[i]
		err = f.Client.Delete(f.Ctx, rbg)
		if err != nil {
			panic(fmt.Sprintf("failed to delete rbg: %s", err.Error()))
		}
	}
}
