package workers_test

import (
	"context"
	"testing"

	"github.com/vela-games/circleci-runner-autoscaler/workers"
	"gotest.tools/v3/assert"
	v1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"
)

func TestK8sDiscoveryWorker(t *testing.T) {
	t.Run("it should only start k8s scaling worker once", func(t *testing.T) {
		k8sClient := testclient.NewSimpleClientset(&v1.CronJobList{
			Items: []v1.CronJob{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "patcher",
						Namespace: "circleci-runners",
						Labels: map[string]string{
							"resource-class-org":  "vela-games",
							"resource-class-name": "k8s-patcher",
						},
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "CronJob",
						APIVersion: "batch/v1",
					},
					Spec: v1.CronJobSpec{},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "other-cronjob-2",
						Namespace: "circleci-runners",
						Labels: map[string]string{
							"resource-class-org":  "unknown-namespace",
							"resource-class-name": "k8s-patcher",
						},
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "CronJob",
						APIVersion: "batch/v1",
					},
					Spec: v1.CronJobSpec{},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "patcher",
						Namespace: "other-namespace",
						Labels: map[string]string{
							"resource-class-org":  "vela-games",
							"resource-class-name": "k8s-patcher",
						},
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "CronJob",
						APIVersion: "batch/v1",
					},
					Spec: v1.CronJobSpec{},
				},
			},
		})

		dispatcher := &WorkerDispatcherTest{}

		discovery := workers.K8sDiscoveryWorker{
			Dispatcher:   dispatcher,
			Namespace:    "vela-games",
			ClientSet:    k8sClient,
			K8sNamespace: "circleci-runners",
		}

		discovery.Handle(context.TODO())
		discovery.Handle(context.TODO())

		assert.Equal(t, 1, dispatcher.Count)
	})

	t.Run("it should start two k8s scaling worker once", func(t *testing.T) {
		k8sClient := testclient.NewSimpleClientset(&v1.CronJobList{
			Items: []v1.CronJob{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "patcher",
						Namespace: "circleci-runners",
						Labels: map[string]string{
							"resource-class-org":  "vela-games",
							"resource-class-name": "k8s-patcher",
						},
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "CronJob",
						APIVersion: "batch/v1",
					},
					Spec: v1.CronJobSpec{},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "other-cronjob-2",
						Namespace: "circleci-runners",
						Labels: map[string]string{
							"resource-class-org":  "unknown-namespace",
							"resource-class-name": "k8s-patcher",
						},
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "CronJob",
						APIVersion: "batch/v1",
					},
					Spec: v1.CronJobSpec{},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "patcher",
						Namespace: "other-namespace",
						Labels: map[string]string{
							"resource-class-org":  "vela-games",
							"resource-class-name": "k8s-patcher",
						},
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "CronJob",
						APIVersion: "batch/v1",
					},
					Spec: v1.CronJobSpec{},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "patcher-2",
						Namespace: "circleci-runners",
						Labels: map[string]string{
							"resource-class-org":  "vela-games",
							"resource-class-name": "k8s-patcher-2",
						},
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "CronJob",
						APIVersion: "batch/v1",
					},
					Spec: v1.CronJobSpec{},
				},
			},
		})

		dispatcher := &WorkerDispatcherTest{}

		discovery := workers.K8sDiscoveryWorker{
			Dispatcher:   dispatcher,
			Namespace:    "vela-games",
			ClientSet:    k8sClient,
			K8sNamespace: "circleci-runners",
		}

		discovery.Handle(context.TODO())
		discovery.Handle(context.TODO())
		discovery.Handle(context.TODO())

		assert.Equal(t, 2, dispatcher.Count)
	})
}
