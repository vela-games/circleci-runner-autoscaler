package workers_test

import (
	"context"
	"net/http"
	"strconv"
	"testing"
	"time"

	circleci_client "github.com/vela-games/circleci-runner-autoscaler/client"
	"github.com/vela-games/circleci-runner-autoscaler/workers"
	"gotest.tools/v3/assert"
	v1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	testclient "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestK8sScalingWorker(t *testing.T) {
	t.Run("it should do nothing", func(t *testing.T) {
		k8sClient := testclient.NewSimpleClientset(&v1.CronJobList{
			Items: []v1.CronJob{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cronjob-class",
						Namespace: "cronjob-namespace",
						Labels: map[string]string{
							"resource-class-org":  "vela-games",
							"resource-class-name": "my-resource-class",
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

		k8sClient.PrependReactor("create", "jobs", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			t.Error("CreateJob was called. This shouldn't be doing anything")
			t.FailNow()
			return false, nil, nil
		})

		k8sClient.PrependReactor("get", "cronjobs", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			t.Error("Get CronJob was called")
			t.FailNow()
			return false, nil, nil
		})

		scaling := &workers.K8sScalingWorker{
			ResourceClass:    "vela-games/my-resource-class",
			CronJobName:      "cronjob-class",
			CronJobNamespace: "cronjob-namespace",
			ClientSet:        k8sClient,
		}

		ciClient := &mockCircleCiClient{
			MockGetUnclaimedTasksWithResponse: func(ctx context.Context, params *circleci_client.GetUnclaimedTasksParams, reqEditors ...circleci_client.RequestEditorFn) (*circleci_client.GetUnclaimedTasksResponse, error) {
				return &circleci_client.GetUnclaimedTasksResponse{
					HTTPResponse: &http.Response{
						StatusCode: 200,
					},
					JSON200: &circleci_client.UnclaimedTaskCount{
						UnclaimedTaskCount: intPointer(0),
					},
				}, nil
			},
			MockGetRunnersWithResponse: func(ctx context.Context, params *circleci_client.GetRunnersParams, reqEditors ...circleci_client.RequestEditorFn) (*circleci_client.GetRunnersResponse, error) {
				t.Error("RunnersWithResponse was called")
				t.FailNow()
				return nil, nil
			},
			MockGetRunningTasksWithResponse: func(ctx context.Context, params *circleci_client.GetRunningTasksParams, reqEditors ...circleci_client.RequestEditorFn) (*circleci_client.GetRunningTasksResponse, error) {
				return nil, nil
			},
		}

		scaling.CircleCiClient = ciClient

		scaling.Handle(context.TODO())
	})

	t.Run("it should create 4 k8s jobs", func(t *testing.T) {
		now := time.Now()
		sec := now.Unix()

		k8sClient := testclient.NewSimpleClientset(&v1.CronJobList{
			Items: []v1.CronJob{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cronjob-class",
						Namespace: "cronjob-namespace",
						Labels: map[string]string{
							"resource-class-org":  "vela-games",
							"resource-class-name": "my-resource-class",
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
		},

			&corev1.PodList{
				Items: []corev1.Pod{
					{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Pod",
							APIVersion: "v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "cronjob-class-" + strconv.FormatInt(sec, 10) + "-1",
							Namespace: "cronjob-namespace",
							Labels: map[string]string{
								"resource-class-org":  "vela-games",
								"resource-class-name": "my-resource-class",
							},
						},
						Spec: corev1.PodSpec{},
						Status: corev1.PodStatus{
							Phase: corev1.PodRunning,
						},
					},
					{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Pod",
							APIVersion: "v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "cronjob-class-" + strconv.FormatInt(sec, 10) + "-2",
							Namespace: "cronjob-namespace",
							Labels: map[string]string{
								"resource-class-org":  "vela-games",
								"resource-class-name": "my-resource-class",
							},
						},
						Spec: corev1.PodSpec{},
						Status: corev1.PodStatus{
							Phase: corev1.PodRunning,
						},
					},
					{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Pod",
							APIVersion: "v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "cronjob-class-" + strconv.FormatInt(sec, 10) + "-3",
							Namespace: "cronjob-namespace",
							Labels: map[string]string{
								"resource-class-org":  "vela-games",
								"resource-class-name": "my-resource-class",
							},
						},
						Spec: corev1.PodSpec{},
						Status: corev1.PodStatus{
							Phase: corev1.PodRunning,
						},
					},
					{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Pod",
							APIVersion: "v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "cronjob-class-" + strconv.FormatInt(sec, 10) + "-4",
							Namespace: "cronjob-namespace",
							Labels: map[string]string{
								"resource-class-org":  "vela-games",
								"resource-class-name": "my-resource-class",
							},
						},
						Spec: corev1.PodSpec{},
						Status: corev1.PodStatus{
							Phase: corev1.PodRunning,
						},
					},
					{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Pod",
							APIVersion: "v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "cronjob-class-other-1",
							Namespace: "cronjob-namespace",
							Labels: map[string]string{
								"resource-class-org":  "vela-games",
								"resource-class-name": "my-resource-class-2",
							},
						},
						Spec: corev1.PodSpec{},
						Status: corev1.PodStatus{
							Phase: corev1.PodSucceeded,
						},
					},
					{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Pod",
							APIVersion: "v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "cronjob-class-other-2",
							Namespace: "cronjob-namespace",
							Labels: map[string]string{
								"resource-class-org":  "vela-games",
								"resource-class-name": "my-resource-class",
							},
						},
						Spec: corev1.PodSpec{},
						Status: corev1.PodStatus{
							Phase: corev1.PodSucceeded,
						},
					},
				},
			})

		podListCount := 0
		jobCreatedCount := 0
		getJobCount := 0

		k8sClient.PrependReactor("list", "pods", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			podListCount++
			return false, nil, nil
		})

		k8sClient.PrependReactor("create", "jobs", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			jobCreatedCount++
			return false, nil, nil
		})

		k8sClient.PrependReactor("get", "cronjobs", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			getJobCount++
			return false, nil, nil
		})

		scaling := &workers.K8sScalingWorker{
			ResourceClass:    "vela-games/my-resource-class",
			CronJobName:      "cronjob-class",
			CronJobNamespace: "cronjob-namespace",
			TimestampGenerator: func() int64 {
				return sec
			},
			ClientSet: k8sClient,
		}

		ciClient := &mockCircleCiClient{
			MockGetUnclaimedTasksWithResponse: func(ctx context.Context, params *circleci_client.GetUnclaimedTasksParams, reqEditors ...circleci_client.RequestEditorFn) (*circleci_client.GetUnclaimedTasksResponse, error) {
				return &circleci_client.GetUnclaimedTasksResponse{
					HTTPResponse: &http.Response{
						StatusCode: 200,
					},
					JSON200: &circleci_client.UnclaimedTaskCount{
						UnclaimedTaskCount: intPointer(4),
					},
				}, nil
			},
			MockGetRunnersWithResponse: func(ctx context.Context, params *circleci_client.GetRunnersParams, reqEditors ...circleci_client.RequestEditorFn) (*circleci_client.GetRunnersResponse, error) {
				assert.Equal(t, *params.ResourceClass, "vela-games/my-resource-class")
				return &circleci_client.GetRunnersResponse{
					HTTPResponse: &http.Response{
						StatusCode: 200,
					},
					JSON200: &circleci_client.AgentList{
						Items: &[]circleci_client.Agent{
							{
								Name: stringPointer("cronjob-class-" + strconv.FormatInt(sec, 10) + "-1"),
							},
							{
								Name: stringPointer("cronjob-class-" + strconv.FormatInt(sec, 10) + "-2"),
							},
							{
								Name: stringPointer("cronjob-class-" + strconv.FormatInt(sec, 10) + "-3"),
							},
							{
								Name: stringPointer("cronjob-class-" + strconv.FormatInt(sec, 10) + "-4"),
							},
							{
								Name: stringPointer("other-thing"),
							},
						},
					},
				}, nil
			},
			MockGetRunningTasksWithResponse: func(ctx context.Context, params *circleci_client.GetRunningTasksParams, reqEditors ...circleci_client.RequestEditorFn) (*circleci_client.GetRunningTasksResponse, error) {
				return nil, nil
			},
		}

		scaling.CircleCiClient = ciClient

		scaling.Handle(context.TODO())

		assert.Equal(t, 4, jobCreatedCount)
		assert.Equal(t, 1, getJobCount)
		assert.Equal(t, 1, podListCount)
	})
}
