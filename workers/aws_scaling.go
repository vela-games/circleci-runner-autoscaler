package workers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	backoff "github.com/cenkalti/backoff/v4"
	circleci_client "github.com/vela-games/circleci-runner-autoscaler/client"
	"github.com/vela-games/circleci-runner-autoscaler/services"
)

type AWSScalingWorker struct {
	ResourceClass string

	AsgAwsService  services.AutoScalingAPI
	CircleCiClient circleci_client.ClientWithResponsesInterface
}

// Handle autoscaling for the ResourceClass defined in the struct
func (w *AWSScalingWorker) Handle(ctx context.Context) {
	log.Printf("handle scaling of %v", w.ResourceClass)

	// Get count of unclaimed tasks
	response, err := w.CircleCiClient.GetUnclaimedTasksWithResponse(ctx, &circleci_client.GetUnclaimedTasksParams{
		ResourceClass: w.ResourceClass,
	})
	if err != nil {
		log.Printf("error getting unclaimed tasks by resource class %v: %v", w.ResourceClass, err)
		return
	}

	if response.StatusCode() != 200 {
		log.Printf("got %v code instead of 200", response.StatusCode())
		return
	}

	unclaimedTaskCount := *response.JSON200.UnclaimedTaskCount
	if unclaimedTaskCount > 0 {
		// Get AutoScalingGroup associated with ResourceClass
		asg, err := w.AsgAwsService.DescribeAutoScalingGroups(ctx, &autoscaling.DescribeAutoScalingGroupsInput{
			AutoScalingGroupNames: []string{
				w.ResourceClass,
			},
		})

		if err != nil {
			log.Printf("error trying to describe ASG %v: %v", w.ResourceClass, err)
			return
		}

		if len(asg.AutoScalingGroups) == 0 {
			log.Printf("AWS api didn't return the ASG %v: %v", w.ResourceClass, err)
			return
		}

		if *asg.AutoScalingGroups[0].DesiredCapacity == *asg.AutoScalingGroups[0].MaxSize {
			log.Printf("resource class ASG %v is at full capacity", w.ResourceClass)
			return
		}

		// Calculate by how much we are going to increase the desired capacity
		// We increase by the max amount unless the amount of unclaimed tasks is less than that, in which case we add instances equivalent to the unclaimed task count.
		increaseDesiredCapacityBy := int32(*asg.AutoScalingGroups[0].DesiredCapacity) + (int32(*asg.AutoScalingGroups[0].MaxSize) - int32(*asg.AutoScalingGroups[0].DesiredCapacity))
		if increaseDesiredCapacityBy > int32(unclaimedTaskCount) {
			increaseDesiredCapacityBy = int32(*asg.AutoScalingGroups[0].DesiredCapacity) + int32(unclaimedTaskCount)
		}

		if increaseDesiredCapacityBy > int32(*asg.AutoScalingGroups[0].MaxSize) {
			increaseDesiredCapacityBy = int32(*asg.AutoScalingGroups[0].MaxSize)
		}

		log.Printf("%v has %v unclaimed tasks, current desired capacity %v, new desired capacity %v", w.ResourceClass, unclaimedTaskCount, *asg.AutoScalingGroups[0].DesiredCapacity, increaseDesiredCapacityBy)

		// Set the desired capacity
		_, err = w.AsgAwsService.SetDesiredCapacity(ctx, &autoscaling.SetDesiredCapacityInput{
			AutoScalingGroupName: &w.ResourceClass,
			DesiredCapacity:      &increaseDesiredCapacityBy,
		})
		if err != nil {
			log.Printf("error setting desired capacity for %v", w.ResourceClass)
			return
		}

		// We check that the instances are running and ready to recieve tasks before exiting the func, as if Handle() get executed immediately after
		// the unclaimed task amount will still be greater than 0 and we add more instances than we need
		checkRunnersAreUp := func() error {
			runners, err := w.CircleCiClient.GetRunnersWithResponse(ctx, &circleci_client.GetRunnersParams{
				ResourceClass: &w.ResourceClass,
			})
			if err != nil {
				log.Printf("error getting runners from circleci for resourceclass %v", w.ResourceClass)
				return err
			}

			if runners.StatusCode() != 200 {
				message := fmt.Sprintf("error getting runners for resourceClass %v from circleci unexpected statusCode %v", w.ResourceClass, runners.StatusCode())
				log.Println(message)
				return errors.New(message)
			}

			asg, err := w.AsgAwsService.DescribeAutoScalingGroups(ctx, &autoscaling.DescribeAutoScalingGroupsInput{
				AutoScalingGroupNames: []string{
					w.ResourceClass,
				},
			})
			if err != nil {
				log.Printf("error getting runners from circleci for resourceclass %v", w.ResourceClass)
				return err
			}

			foundCount := 0
			for _, instance := range asg.AutoScalingGroups[0].Instances {
				// We only check instances 'InService' as to try to avoid as scenario where a runner is being terminated while we are creating a new one
				if instance.LifecycleState != "InService" {
					continue
				}
				for _, runner := range *runners.JSON200.Items {
					if *runner.Name == *instance.InstanceId {
						foundCount++
					}
				}
			}

			if int32(foundCount) != *asg.AutoScalingGroups[0].DesiredCapacity {
				return errors.New("waiting for all runners to come up")
			}

			return nil
		}

		// We use exponential backoff to retry the check as instances take a couple of minutes to come up
		err = backoff.RetryNotify(checkRunnersAreUp, backoff.NewExponentialBackOff(), func(e error, d time.Duration) {
			if e != nil {
				log.Printf("%v: %v retry in %v", w.ResourceClass, e.Error(), d.String())
			}

		})
		if err != nil {
			log.Printf("%v: unrecoverable error %v", w.ResourceClass, err)
			return
		}

	} else {
		log.Printf("%v: no unclaimed tasks", w.ResourceClass)
	}
}
