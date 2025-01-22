package util

import (
	"context"
	"time"

	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

type PodSpec struct {
	Name        string
	Node        string
	Req         v1.ResourceList
	Tolerations []v1.Toleration
}

func CreatePod(ctx *TestContext, spec PodSpec) *v1.Pod {
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      spec.Name,
			Namespace: ctx.Namespace,
		},
		Spec: v1.PodSpec{
			NodeName: spec.Node,
			Containers: []v1.Container{
				{
					Image:           DefaultNginxImage,
					Name:            spec.Name,
					ImagePullPolicy: v1.PullIfNotPresent,
					Resources: v1.ResourceRequirements{
						Requests: spec.Req,
					},
				},
			},
			Tolerations: spec.Tolerations,
		},
	}

	pod, err := ctx.Kubeclient.CoreV1().Pods(ctx.Namespace).Create(context.TODO(), pod, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred(), "failed to create pod %s", spec.Name)

	return pod
}

func WaitPodReady(ctx *TestContext, pod *v1.Pod) error {
	return wait.Poll(100*time.Millisecond, FiveMinute, func() (bool, error) {
		pod, err := ctx.Kubeclient.CoreV1().Pods(pod.Namespace).Get(context.TODO(), pod.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		return pod.Status.Phase == v1.PodRunning, nil
	})
}
