package workers_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling/types"
	circleci_client "github.com/vela-games/circleci-runner-autoscaler/client"
	"github.com/vela-games/circleci-runner-autoscaler/workers"
	"gotest.tools/v3/assert"
)

func intPointer(i int) *int {
	return &i
}

func int32Pointer(i int32) *int32 {
	return &i
}

type mockGetRunnersWithResponse func(ctx context.Context, params *circleci_client.GetRunnersParams, reqEditors ...circleci_client.RequestEditorFn) (*circleci_client.GetRunnersResponse, error)
type mockGetUnclaimedTasksWithResponse func(ctx context.Context, params *circleci_client.GetUnclaimedTasksParams, reqEditors ...circleci_client.RequestEditorFn) (*circleci_client.GetUnclaimedTasksResponse, error)
type mockGetRunningTasksWithResponse func(ctx context.Context, params *circleci_client.GetRunningTasksParams, reqEditors ...circleci_client.RequestEditorFn) (*circleci_client.GetRunningTasksResponse, error)

type mockCircleCiClient struct {
	MockGetRunnersWithResponse        mockGetRunnersWithResponse
	MockGetUnclaimedTasksWithResponse mockGetUnclaimedTasksWithResponse
	MockGetRunningTasksWithResponse   mockGetRunningTasksWithResponse
}

func (m *mockCircleCiClient) GetRunnersWithResponse(ctx context.Context, params *circleci_client.GetRunnersParams, reqEditors ...circleci_client.RequestEditorFn) (*circleci_client.GetRunnersResponse, error) {
	return m.MockGetRunnersWithResponse(ctx, params, reqEditors...)
}

func (m *mockCircleCiClient) GetUnclaimedTasksWithResponse(ctx context.Context, params *circleci_client.GetUnclaimedTasksParams, reqEditors ...circleci_client.RequestEditorFn) (*circleci_client.GetUnclaimedTasksResponse, error) {
	return m.MockGetUnclaimedTasksWithResponse(ctx, params, reqEditors...)
}

func (m *mockCircleCiClient) GetRunningTasksWithResponse(ctx context.Context, params *circleci_client.GetRunningTasksParams, reqEditors ...circleci_client.RequestEditorFn) (*circleci_client.GetRunningTasksResponse, error) {
	return m.MockGetRunningTasksWithResponse(ctx, params, reqEditors...)
}

func TestAWSScalingWorker(t *testing.T) {
	scaling := &workers.AWSScalingWorker{
		ResourceClass: "vela-games/my-resource-class",
	}

	t.Run("it should do nothing", func(t *testing.T) {
		asgClient := &mockAutoScalingGroupsAPI{
			MockDescribeAutoScalingGroupsAPI: func(ctx context.Context, params *autoscaling.DescribeAutoScalingGroupsInput, optFns ...func(*autoscaling.Options)) (*autoscaling.DescribeAutoScalingGroupsOutput, error) {
				t.Error("DescribeAutoScalingGroups was called")
				return nil, nil
			},
			MockSetDesiredCapacityAPI: func(ctx context.Context, params *autoscaling.SetDesiredCapacityInput, optFns ...func(*autoscaling.Options)) (*autoscaling.SetDesiredCapacityOutput, error) {
				t.Error("SetDesiredCapacity was called")
				return nil, nil
			},
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
				return nil, nil
			},
			MockGetRunningTasksWithResponse: func(ctx context.Context, params *circleci_client.GetRunningTasksParams, reqEditors ...circleci_client.RequestEditorFn) (*circleci_client.GetRunningTasksResponse, error) {
				return nil, nil
			},
		}

		scaling.CircleCiClient = ciClient
		scaling.AsgAwsService = asgClient

		scaling.Handle(context.TODO())
	})

	t.Run("it should do nothing becuase ASG DesiredCapacity==MaxSize", func(t *testing.T) {
		asgClient := &mockAutoScalingGroupsAPI{
			MockDescribeAutoScalingGroupsAPI: func(ctx context.Context, params *autoscaling.DescribeAutoScalingGroupsInput, optFns ...func(*autoscaling.Options)) (*autoscaling.DescribeAutoScalingGroupsOutput, error) {
				return &autoscaling.DescribeAutoScalingGroupsOutput{
					AutoScalingGroups: []types.AutoScalingGroup{
						{
							AutoScalingGroupName: stringPointer("vela-games/my-resource-class"),
							DesiredCapacity:      int32Pointer(10),
							MaxSize:              int32Pointer(10),
							MinSize:              int32Pointer(0),
						},
					},
				}, nil
			},
			MockSetDesiredCapacityAPI: func(ctx context.Context, params *autoscaling.SetDesiredCapacityInput, optFns ...func(*autoscaling.Options)) (*autoscaling.SetDesiredCapacityOutput, error) {
				t.Error("SetDesiredCapacity was called")
				return nil, nil
			},
		}

		ciClient := &mockCircleCiClient{
			MockGetUnclaimedTasksWithResponse: func(ctx context.Context, params *circleci_client.GetUnclaimedTasksParams, reqEditors ...circleci_client.RequestEditorFn) (*circleci_client.GetUnclaimedTasksResponse, error) {
				return &circleci_client.GetUnclaimedTasksResponse{
					HTTPResponse: &http.Response{
						StatusCode: 200,
					},
					JSON200: &circleci_client.UnclaimedTaskCount{
						UnclaimedTaskCount: intPointer(1),
					},
				}, nil
			},
			MockGetRunnersWithResponse: func(ctx context.Context, params *circleci_client.GetRunnersParams, reqEditors ...circleci_client.RequestEditorFn) (*circleci_client.GetRunnersResponse, error) {
				return nil, nil
			},
			MockGetRunningTasksWithResponse: func(ctx context.Context, params *circleci_client.GetRunningTasksParams, reqEditors ...circleci_client.RequestEditorFn) (*circleci_client.GetRunningTasksResponse, error) {
				return nil, nil
			},
		}

		scaling.CircleCiClient = ciClient
		scaling.AsgAwsService = asgClient

		scaling.Handle(context.TODO())
	})

	t.Run("it should increase ASG DesiredCapacity by 4", func(t *testing.T) {
		callCount := 0

		asgClient := &mockAutoScalingGroupsAPI{
			MockDescribeAutoScalingGroupsAPI: func(ctx context.Context, params *autoscaling.DescribeAutoScalingGroupsInput, optFns ...func(*autoscaling.Options)) (*autoscaling.DescribeAutoScalingGroupsOutput, error) {
				if callCount == 0 {
					callCount++
					return &autoscaling.DescribeAutoScalingGroupsOutput{
						AutoScalingGroups: []types.AutoScalingGroup{
							{
								AutoScalingGroupName: stringPointer("vela-games/my-resource-class"),
								DesiredCapacity:      int32Pointer(1),
								MaxSize:              int32Pointer(10),
								MinSize:              int32Pointer(0),
							},
						},
					}, nil
				} else {
					return &autoscaling.DescribeAutoScalingGroupsOutput{
						AutoScalingGroups: []types.AutoScalingGroup{
							{
								AutoScalingGroupName: stringPointer("vela-games/my-resource-class"),
								DesiredCapacity:      int32Pointer(5),
								MaxSize:              int32Pointer(10),
								MinSize:              int32Pointer(0),
								Instances: []types.Instance{
									{
										InstanceId:     stringPointer("i-laiCh3oo"),
										LifecycleState: "InService",
									},
									{
										InstanceId:     stringPointer("i-As0iugan"),
										LifecycleState: "InService",
									},
									{
										InstanceId:     stringPointer("i-Qui6josh"),
										LifecycleState: "InService",
									},
									{
										InstanceId:     stringPointer("i-EeSe5Tha"),
										LifecycleState: "InService",
									},
									{
										InstanceId:     stringPointer("i-ool0jooM"),
										LifecycleState: "InService",
									},
								},
							},
						},
					}, nil
				}

			},
			MockSetDesiredCapacityAPI: func(ctx context.Context, params *autoscaling.SetDesiredCapacityInput, optFns ...func(*autoscaling.Options)) (*autoscaling.SetDesiredCapacityOutput, error) {
				assert.Equal(t, *params.AutoScalingGroupName, "vela-games/my-resource-class")
				assert.Equal(t, *params.DesiredCapacity, int32(5))
				return nil, nil
			},
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
								Name: stringPointer("i-laiCh3oo"),
							},
							{
								Name: stringPointer("i-As0iugan"),
							},
							{
								Name: stringPointer("i-Qui6josh"),
							},
							{
								Name: stringPointer("i-EeSe5Tha"),
							},
							{
								Name: stringPointer("i-ool0jooM"),
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
		scaling.AsgAwsService = asgClient

		scaling.Handle(context.TODO())
	})

	t.Run("it should increase ASG DesiredCapacity by 2 reaching max", func(t *testing.T) {
		callCount := 0

		asgClient := &mockAutoScalingGroupsAPI{
			MockDescribeAutoScalingGroupsAPI: func(ctx context.Context, params *autoscaling.DescribeAutoScalingGroupsInput, optFns ...func(*autoscaling.Options)) (*autoscaling.DescribeAutoScalingGroupsOutput, error) {
				if callCount == 0 {
					callCount++
					return &autoscaling.DescribeAutoScalingGroupsOutput{
						AutoScalingGroups: []types.AutoScalingGroup{
							{
								AutoScalingGroupName: stringPointer("vela-games/my-resource-class"),
								DesiredCapacity:      int32Pointer(8),
								MaxSize:              int32Pointer(10),
								MinSize:              int32Pointer(0),
							},
						},
					}, nil
				} else {
					return &autoscaling.DescribeAutoScalingGroupsOutput{
						AutoScalingGroups: []types.AutoScalingGroup{
							{
								AutoScalingGroupName: stringPointer("vela-games/my-resource-class"),
								DesiredCapacity:      int32Pointer(10),
								MaxSize:              int32Pointer(10),
								MinSize:              int32Pointer(0),
								Instances: []types.Instance{
									{
										InstanceId:     stringPointer("i-laiCh3oo"),
										LifecycleState: "InService",
									},
									{
										InstanceId:     stringPointer("i-As0iugan"),
										LifecycleState: "InService",
									},
									{
										InstanceId:     stringPointer("i-Qui6josh"),
										LifecycleState: "InService",
									},
									{
										InstanceId:     stringPointer("i-EeSe5Tha"),
										LifecycleState: "InService",
									},
									{
										InstanceId:     stringPointer("i-ool0jooM"),
										LifecycleState: "InService",
									},
									{
										InstanceId:     stringPointer("i-laiCh3oo1"),
										LifecycleState: "InService",
									},
									{
										InstanceId:     stringPointer("i-laiCh3oo2"),
										LifecycleState: "InService",
									},
									{
										InstanceId:     stringPointer("i-laiCh3oo3"),
										LifecycleState: "InService",
									},
									{
										InstanceId:     stringPointer("i-laiCh3oo4"),
										LifecycleState: "InService",
									},
									{
										InstanceId:     stringPointer("i-laiCh3oo5"),
										LifecycleState: "InService",
									},
								},
							},
						},
					}, nil
				}

			},
			MockSetDesiredCapacityAPI: func(ctx context.Context, params *autoscaling.SetDesiredCapacityInput, optFns ...func(*autoscaling.Options)) (*autoscaling.SetDesiredCapacityOutput, error) {
				assert.Equal(t, *params.AutoScalingGroupName, "vela-games/my-resource-class")
				assert.Equal(t, *params.DesiredCapacity, int32(10))
				return nil, nil
			},
		}

		ciClient := &mockCircleCiClient{
			MockGetUnclaimedTasksWithResponse: func(ctx context.Context, params *circleci_client.GetUnclaimedTasksParams, reqEditors ...circleci_client.RequestEditorFn) (*circleci_client.GetUnclaimedTasksResponse, error) {
				return &circleci_client.GetUnclaimedTasksResponse{
					HTTPResponse: &http.Response{
						StatusCode: 200,
					},
					JSON200: &circleci_client.UnclaimedTaskCount{
						UnclaimedTaskCount: intPointer(2),
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
								Name: stringPointer("i-laiCh3oo"),
							},
							{
								Name: stringPointer("i-As0iugan"),
							},
							{
								Name: stringPointer("i-Qui6josh"),
							},
							{
								Name: stringPointer("i-EeSe5Tha"),
							},
							{
								Name: stringPointer("i-ool0jooM"),
							},
							{
								Name: stringPointer("i-laiCh3oo1"),
							},
							{
								Name: stringPointer("i-laiCh3oo2"),
							},
							{
								Name: stringPointer("i-laiCh3oo3"),
							},
							{
								Name: stringPointer("i-laiCh3oo4"),
							},
							{
								Name: stringPointer("i-laiCh3oo5"),
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
		scaling.AsgAwsService = asgClient

		scaling.Handle(context.TODO())
	})

	t.Run("it should increase ASG to max", func(t *testing.T) {
		callCount := 0

		asgClient := &mockAutoScalingGroupsAPI{
			MockDescribeAutoScalingGroupsAPI: func(ctx context.Context, params *autoscaling.DescribeAutoScalingGroupsInput, optFns ...func(*autoscaling.Options)) (*autoscaling.DescribeAutoScalingGroupsOutput, error) {
				if callCount == 0 {
					callCount++
					return &autoscaling.DescribeAutoScalingGroupsOutput{
						AutoScalingGroups: []types.AutoScalingGroup{
							{
								AutoScalingGroupName: stringPointer("vela-games/my-resource-class"),
								DesiredCapacity:      int32Pointer(1),
								MaxSize:              int32Pointer(10),
								MinSize:              int32Pointer(0),
							},
						},
					}, nil
				} else {
					return &autoscaling.DescribeAutoScalingGroupsOutput{
						AutoScalingGroups: []types.AutoScalingGroup{
							{
								AutoScalingGroupName: stringPointer("vela-games/my-resource-class"),
								DesiredCapacity:      int32Pointer(10),
								MaxSize:              int32Pointer(10),
								MinSize:              int32Pointer(0),
								Instances: []types.Instance{
									{
										InstanceId:     stringPointer("i-laiCh3oo"),
										LifecycleState: "InService",
									},
									{
										InstanceId:     stringPointer("i-As0iugan"),
										LifecycleState: "InService",
									},
									{
										InstanceId:     stringPointer("i-Qui6josh"),
										LifecycleState: "InService",
									},
									{
										InstanceId:     stringPointer("i-EeSe5Tha"),
										LifecycleState: "InService",
									},
									{
										InstanceId:     stringPointer("i-ool0jooM"),
										LifecycleState: "InService",
									},
									{
										InstanceId:     stringPointer("i-laiCh3oo1"),
										LifecycleState: "InService",
									},
									{
										InstanceId:     stringPointer("i-laiCh3oo2"),
										LifecycleState: "InService",
									},
									{
										InstanceId:     stringPointer("i-laiCh3oo3"),
										LifecycleState: "InService",
									},
									{
										InstanceId:     stringPointer("i-laiCh3oo4"),
										LifecycleState: "InService",
									},
									{
										InstanceId:     stringPointer("i-laiCh3oo5"),
										LifecycleState: "InService",
									},
								},
							},
						},
					}, nil
				}

			},
			MockSetDesiredCapacityAPI: func(ctx context.Context, params *autoscaling.SetDesiredCapacityInput, optFns ...func(*autoscaling.Options)) (*autoscaling.SetDesiredCapacityOutput, error) {
				assert.Equal(t, *params.AutoScalingGroupName, "vela-games/my-resource-class")
				assert.Equal(t, *params.DesiredCapacity, int32(10))
				return nil, nil
			},
		}

		ciClient := &mockCircleCiClient{
			MockGetUnclaimedTasksWithResponse: func(ctx context.Context, params *circleci_client.GetUnclaimedTasksParams, reqEditors ...circleci_client.RequestEditorFn) (*circleci_client.GetUnclaimedTasksResponse, error) {
				return &circleci_client.GetUnclaimedTasksResponse{
					HTTPResponse: &http.Response{
						StatusCode: 200,
					},
					JSON200: &circleci_client.UnclaimedTaskCount{
						UnclaimedTaskCount: intPointer(15),
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
								Name: stringPointer("i-laiCh3oo"),
							},
							{
								Name: stringPointer("i-As0iugan"),
							},
							{
								Name: stringPointer("i-Qui6josh"),
							},
							{
								Name: stringPointer("i-EeSe5Tha"),
							},
							{
								Name: stringPointer("i-ool0jooM"),
							},
							{
								Name: stringPointer("i-laiCh3oo1"),
							},
							{
								Name: stringPointer("i-laiCh3oo2"),
							},
							{
								Name: stringPointer("i-laiCh3oo3"),
							},
							{
								Name: stringPointer("i-laiCh3oo4"),
							},
							{
								Name: stringPointer("i-laiCh3oo5"),
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
		scaling.AsgAwsService = asgClient

		scaling.Handle(context.TODO())
	})

}
