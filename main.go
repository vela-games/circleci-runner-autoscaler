package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	ci_client "github.com/vela-games/circleci-runner-autoscaler/client"
	"github.com/vela-games/circleci-runner-autoscaler/workers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	_ "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"

	"github.com/deepmap/oapi-codegen/pkg/securityprovider"
	"golang.org/x/sync/errgroup"
)

func main() {
	var err error

	asgAwsService, err := initAwsService()
	if err != nil {
		log.Fatalf("unable to initialize AWS SDK, %v", err)
	}

	circleCiClient, err := initCircleCIClient()
	if err != nil {
		log.Fatalf("unable to initialize CircleCI Client: %v", err)
	}

	k8sClient, err := initK8sClient()
	if err != nil {
		log.Fatalf("unable to initialize k8s Client: %v", err)
	}

	group, ctx := errgroup.WithContext(context.Background())
	workerDispatcher := &workers.WorkerDispatcher{
		RunEvery: 5 * time.Second,
		Group:    group,
	}

	awsDiscoveryWorker := &workers.AWSDiscoveryWorker{
		Namespace:      "vela-games",
		AsgAwsService:  asgAwsService,
		CircleCiClient: circleCiClient,
		Dispatcher:     workerDispatcher,
	}
	workerDispatcher.Start(ctx, awsDiscoveryWorker)

	k8sDiscoveryWorker := &workers.K8sDiscoveryWorker{
		Namespace:      "vela-games",
		K8sNamespace:   "circleci-runners",
		ClientSet:      k8sClient,
		CircleCiClient: circleCiClient,
		Dispatcher:     workerDispatcher,
	}
	workerDispatcher.Start(ctx, k8sDiscoveryWorker)

	subscribeToSyscallSignal(group)
	if err := group.Wait(); err != nil {
		log.Printf("%v", err)
	}
}

func subscribeToSyscallSignal(group *errgroup.Group) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	group.Go(func() error {
		<-sigs
		return fmt.Errorf("syscall signal recieved. exiting")
	})
}

func initK8sClient() (*kubernetes.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return clientset, nil
}

func initAwsService() (*autoscaling.Client, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, err
	}

	return autoscaling.NewFromConfig(cfg), nil
}

func initCircleCIClient() (*ci_client.ClientWithResponses, error) {
	circleToken := os.Getenv("CIRCLE_TOKEN")
	if len(circleToken) == 0 {
		return nil, fmt.Errorf("the environment variable CIRCLE_TOKEN has to be defined")
	}

	apiKeyProvider, err := securityprovider.NewSecurityProviderApiKey("header", "Circle-Token", circleToken)
	if err != nil {
		return nil, err
	}

	client, err := ci_client.NewClientWithResponses("https://runner.circleci.com/api/v2", ci_client.WithRequestEditorFn(apiKeyProvider.Intercept))
	if err != nil {
		return nil, err
	}

	return client, nil
}
