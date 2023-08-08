package workers

import (
	"context"
	"log"
	"time"

	"github.com/vela-games/circleci-runner-autoscaler/client"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type K8sDiscoveryWorker struct {
	Dispatcher Dispatcher

	ClientSet      kubernetes.Interface
	CircleCiClient client.ClientWithResponsesInterface

	K8sNamespace              string
	Namespace                 string
	childWorkersResourceClass []string
}

// This will discover new k8s resource classes on circleci and start the k8s scaling worker for each one of them
func (w *K8sDiscoveryWorker) Handle(ctx context.Context) {
	cronJobList, err := w.ClientSet.BatchV1().CronJobs(w.K8sNamespace).List(ctx, v1.ListOptions{})
	if err != nil {
		log.Printf("error listing cronjobs in namespace: %v, %v", w.K8sNamespace, err)
		return
	}

	for _, job := range cronJobList.Items {
		namespace, ok := job.Labels["resource-class-org"]
		if !ok || namespace != w.Namespace {
			continue
		}

		name, ok := job.Labels["resource-class-name"]
		if !ok {
			continue
		}

		fullClassName := namespace + "/" + name

		found := false
		for _, c := range w.childWorkersResourceClass {
			if fullClassName == c {
				found = true
				break
			}
		}

		if !found {
			w.childWorkersResourceClass = append(w.childWorkersResourceClass, fullClassName)
			log.Printf("Found new k8s resource class %v, starting scaling worker for it", fullClassName)
			sc := &K8sScalingWorker{
				ResourceClass:    fullClassName,
				CronJobNamespace: job.Namespace,
				CronJobName:      job.Name,
				ClientSet:        w.ClientSet,
				CircleCiClient:   w.CircleCiClient,
				TimestampGenerator: func() int64 {
					return time.Now().Unix()
				},
			}
			w.Dispatcher.Start(ctx, sc)
		}

	}
}
