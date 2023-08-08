package workers_test

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling/types"
	"github.com/vela-games/circleci-runner-autoscaler/workers"
	"gotest.tools/v3/assert"
)

func stringPointer(s string) *string {
	return &s
}

type WorkerDispatcherTest struct {
	Count int
}

func (d *WorkerDispatcherTest) Start(ctx context.Context, w workers.Worker) {
	d.Count = d.Count + 1
}

type mockDescribeAutoScalingGroupsAPI func(ctx context.Context, params *autoscaling.DescribeAutoScalingGroupsInput, optFns ...func(*autoscaling.Options)) (*autoscaling.DescribeAutoScalingGroupsOutput, error)
type mockSetDesiredCapacityAPI func(ctx context.Context, params *autoscaling.SetDesiredCapacityInput, optFns ...func(*autoscaling.Options)) (*autoscaling.SetDesiredCapacityOutput, error)

type mockAutoScalingGroupsAPI struct {
	MockDescribeAutoScalingGroupsAPI mockDescribeAutoScalingGroupsAPI
	MockSetDesiredCapacityAPI        mockSetDesiredCapacityAPI
}

func (m mockAutoScalingGroupsAPI) DescribeAutoScalingGroups(ctx context.Context, params *autoscaling.DescribeAutoScalingGroupsInput, optFns ...func(*autoscaling.Options)) (*autoscaling.DescribeAutoScalingGroupsOutput, error) {
	return m.MockDescribeAutoScalingGroupsAPI(ctx, params, optFns...)
}

func (m mockAutoScalingGroupsAPI) SetDesiredCapacity(ctx context.Context, params *autoscaling.SetDesiredCapacityInput, optFns ...func(*autoscaling.Options)) (*autoscaling.SetDesiredCapacityOutput, error) {
	return m.MockSetDesiredCapacityAPI(ctx, params, optFns...)
}

func TestAWSDiscoveryWorker(t *testing.T) {
	t.Run("it should only start scaling worker once", func(t *testing.T) {
		dispatcher := &WorkerDispatcherTest{}

		discovery := workers.AWSDiscoveryWorker{
			Dispatcher: dispatcher,
			Namespace:  "vela-games",
		}

		asgClient := mockAutoScalingGroupsAPI{
			MockDescribeAutoScalingGroupsAPI: func(ctx context.Context, params *autoscaling.DescribeAutoScalingGroupsInput, optFns ...func(*autoscaling.Options)) (*autoscaling.DescribeAutoScalingGroupsOutput, error) {
				return &autoscaling.DescribeAutoScalingGroupsOutput{
					AutoScalingGroups: []types.AutoScalingGroup{
						{
							AutoScalingGroupName: stringPointer("autoscaling-group-1"),
							Tags: []types.TagDescription{
								{
									Key:   stringPointer("resource-class"),
									Value: stringPointer("vela-games/resource-class"),
								},
							},
						},
					},
				}, nil
			},
			MockSetDesiredCapacityAPI: func(ctx context.Context, params *autoscaling.SetDesiredCapacityInput, optFns ...func(*autoscaling.Options)) (*autoscaling.SetDesiredCapacityOutput, error) {
				return nil, nil
			},
		}

		discovery.AsgAwsService = asgClient

		discovery.Handle(context.TODO())
		discovery.Handle(context.TODO())

		assert.Equal(t, 1, dispatcher.Count)
	})

	t.Run("it should start two scaling runners", func(t *testing.T) {
		dispatcher := &WorkerDispatcherTest{}

		discovery := workers.AWSDiscoveryWorker{
			Dispatcher: dispatcher,
			Namespace:  "vela-games",
		}

		asgClient := mockAutoScalingGroupsAPI{
			MockDescribeAutoScalingGroupsAPI: func(ctx context.Context, params *autoscaling.DescribeAutoScalingGroupsInput, optFns ...func(*autoscaling.Options)) (*autoscaling.DescribeAutoScalingGroupsOutput, error) {
				return &autoscaling.DescribeAutoScalingGroupsOutput{
					AutoScalingGroups: []types.AutoScalingGroup{
						{
							AutoScalingGroupName: stringPointer("autoscaling-group-1"),
							Tags: []types.TagDescription{
								{
									Key:   stringPointer("resource-class"),
									Value: stringPointer("vela-games/resource-class"),
								},
							},
						},
						{
							AutoScalingGroupName: stringPointer("autoscaling-group-2"),
							Tags: []types.TagDescription{
								{
									Key:   stringPointer("resource-class"),
									Value: stringPointer("vela-games/resource-class-2"),
								},
							},
						},
					},
				}, nil
			},
			MockSetDesiredCapacityAPI: func(ctx context.Context, params *autoscaling.SetDesiredCapacityInput, optFns ...func(*autoscaling.Options)) (*autoscaling.SetDesiredCapacityOutput, error) {
				return nil, nil
			},
		}

		discovery.AsgAwsService = asgClient

		discovery.Handle(context.TODO())
		assert.Equal(t, 2, dispatcher.Count)
	})

}
