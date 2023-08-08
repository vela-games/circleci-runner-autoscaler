package workers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	backoff "github.com/cenkalti/backoff/v4"
	circleci_client "github.com/vela-games/circleci-runner-autoscaler/client"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

type K8sScalingWorker struct {
	ResourceClass string

	CronJobName      string
	CronJobNamespace string

	TimestampGenerator func() int64

	ClientSet      kubernetes.Interface
	CircleCiClient circleci_client.ClientWithResponsesInterface
}

// Handle autoscaling for the ResourceClass defined in the struct
func (w *K8sScalingWorker) Handle(ctx context.Context) {
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
		// Get CronJob associated with ResourceClass
		cronJob, err := w.ClientSet.BatchV1().CronJobs(w.CronJobNamespace).Get(ctx, w.CronJobName, v1.GetOptions{})
		if err != nil {
			log.Printf("error trying to get CronJob %v: %v", w.ResourceClass, err)
			return
		}

		var jobs []*batchv1.Job

		timestamp := w.TimestampGenerator()

		for i := 0; i < unclaimedTaskCount; i++ {
			job := &batchv1.Job{
				TypeMeta: v1.TypeMeta{
					APIVersion: "batch/v1",
					Kind:       "Job",
				},
				ObjectMeta: v1.ObjectMeta{
					Name:      cronJob.Name + "-" + strconv.FormatInt(timestamp, 10) + "-" + strconv.Itoa(i),
					Namespace: cronJob.Namespace,
					OwnerReferences: []v1.OwnerReference{
						{
							APIVersion: "batch/v1",
							Kind:       "CronJob",
							Name:       cronJob.Name,
							UID:        cronJob.UID,
						},
					},
					Annotations: map[string]string{
						"cronjob.kubernetes.io/instantiate": "manual",
					},
				},
				Spec: cronJob.Spec.JobTemplate.Spec,
			}
			jobs = append(jobs, job)
		}

		log.Printf("%v has %v unclaimed tasks creating k8s jobs", w.ResourceClass, unclaimedTaskCount)

		for _, job := range jobs {
			_, err := w.ClientSet.BatchV1().Jobs(job.Namespace).Create(ctx, job, v1.CreateOptions{})
			if err != nil {
				log.Printf("error creating job %v for %v: %v", job.Name, w.ResourceClass, err)
			}
		}

		// We check that the pods are running and ready to recieve tasks before exiting the func, as if Handle() get executed immediately after
		// the unclaimed task amount will still be greater than 0 and we add more pods than we need
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

			labelMap, _ := v1.LabelSelectorAsMap(&v1.LabelSelector{
				MatchLabels: map[string]string{
					"resource-class-org":  cronJob.Labels["resource-class-org"],
					"resource-class-name": cronJob.Labels["resource-class-name"],
				},
			})

			podList, err := w.ClientSet.CoreV1().Pods(w.CronJobNamespace).List(ctx, v1.ListOptions{
				LabelSelector: labels.SelectorFromSet(labelMap).String(),
			})

			if err != nil {
				log.Printf("error getting pod runners for resourceclass %v", w.ResourceClass)
				return err
			}

			if len(podList.Items) == 0 {
				log.Printf("no pods up yet for %v", w.ResourceClass)
				return errors.New("no pods are up yet")
			}

			targetCount := len(podList.Items)

			foundCount := 0
			for _, pod := range podList.Items {

				// We only check pods in 'Running' state
				if pod.Status.Phase != "Running" {
					targetCount--
					continue
				}

				for _, runner := range *runners.JSON200.Items {
					if *runner.Name == pod.Name {
						foundCount++
					}
				}
			}

			if foundCount == 0 || (foundCount < targetCount) {
				return errors.New("waiting for all runners to come up")
			}

			return nil
		}

		b := &backoff.ExponentialBackOff{
			InitialInterval:     backoff.DefaultInitialInterval,
			RandomizationFactor: backoff.DefaultRandomizationFactor,
			Multiplier:          backoff.DefaultMultiplier,
			MaxInterval:         backoff.DefaultMaxInterval,
			MaxElapsedTime:      1 * time.Minute,
			Stop:                backoff.Stop,
			Clock:               backoff.SystemClock,
		}

		// We use exponential backoff to retry the check as instances take a couple of minutes to come up
		err = backoff.RetryNotify(checkRunnersAreUp, b, func(e error, d time.Duration) {
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
