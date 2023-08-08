package workers

import (
	"context"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/vela-games/circleci-runner-autoscaler/client"
	"github.com/vela-games/circleci-runner-autoscaler/services"
)

type AWSDiscoveryWorker struct {
	Dispatcher Dispatcher

	AsgAwsService  services.AutoScalingAPI
	CircleCiClient client.ClientWithResponsesInterface

	Namespace                 string
	childWorkersResourceClass []string
}

// This will discover new resource classes on circleci and start the scaling worker for each one of them
func (w *AWSDiscoveryWorker) Handle(ctx context.Context) {

	// Get all autoscaling groups on AWS account
	asg, err := w.AsgAwsService.DescribeAutoScalingGroups(ctx, &autoscaling.DescribeAutoScalingGroupsInput{})
	if err != nil {
		log.Printf("error getting autoscaling groups: %v", err)
		return
	}

	// Loop over all ASGs and check their Tags. If the ASG is related to a CircleCI's resource class,
	// it should have the 'resource-class' tag
	for _, asg := range asg.AutoScalingGroups {
		for _, tag := range asg.Tags {
			if *tag.Key == "resource-class" {
				className := *tag.Value

				namespace := strings.Split(className, "/")[0]
				if namespace != w.Namespace {
					continue
				}

				// Check if we already have a worker for this resource class,
				// if not then we start a new scaling worker.
				found := false
				for _, c := range w.childWorkersResourceClass {
					if className == c {
						found = true
						break
					}
				}

				if !found {
					w.childWorkersResourceClass = append(w.childWorkersResourceClass, className)
					log.Printf("Found new resource class %v, starting scaling worker for it", className)
					sc := &AWSScalingWorker{
						ResourceClass:  className,
						AsgAwsService:  w.AsgAwsService,
						CircleCiClient: w.CircleCiClient,
					}
					w.Dispatcher.Start(ctx, sc)
				}
			}
		}
	}
}
