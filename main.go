package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	autoscaler_config "github.com/vela-games/circleci-runner-autoscaler/config"

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
	config, err := autoscaler_config.GetConfig()
	if err != nil {
		log.Panicf("cannot get configuration: %v", err)
	}

	group, ctx := errgroup.WithContext(context.Background())

	asgAwsService, err := initAwsService(ctx)
	if err != nil {
		log.Fatalf("unable to initialize AWS SDK, %v", err)
	}

	circleCiClient, err := initCircleCIClient(config.CircleToken)
	if err != nil {
		log.Fatalf("unable to initialize CircleCI Client: %v", err)
	}

	workerDispatcher := &workers.WorkerDispatcher{
		RunEvery: 5 * time.Second,
		Group:    group,
	}

	awsDiscoveryWorker := &workers.AWSDiscoveryWorker{
		Namespace:      config.CircleResourceNamespace,
		AsgAwsService:  asgAwsService,
		CircleCiClient: circleCiClient,
		Dispatcher:     workerDispatcher,
	}
	workerDispatcher.Start(ctx, awsDiscoveryWorker)

	if config.KubernetesScalerEnabled {
		k8sClient, err := initK8sClient()
		if err != nil {
			log.Fatalf("unable to initialize k8s Client: %v", err)
		}

		k8sDiscoveryWorker := &workers.K8sDiscoveryWorker{
			Namespace:      config.CircleResourceNamespace,
			K8sNamespace:   config.KubernetesNamespace,
			ClientSet:      k8sClient,
			CircleCiClient: circleCiClient,
			Dispatcher:     workerDispatcher,
		}
		workerDispatcher.Start(ctx, k8sDiscoveryWorker)
	}

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

func initAwsService(ctx context.Context) (*autoscaling.Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}

	return autoscaling.NewFromConfig(cfg), nil
}

func initCircleCIClient(circleToken string) (*ci_client.ClientWithResponses, error) {
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

func lookupEnvOrString(key string, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return defaultVal
}
