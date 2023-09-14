package tests

import (
	"context"
	"fmt"

	"github.com/kubearmor/KubeArmor/tests/util"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("intigration", func() {

	Describe("Running Intigrations test", func() {
		It("does syslog get logs", func() {
			k8sClient := util.GetK8sClient()

			podSelector := metav1.ListOptions{
				LabelSelector: "group=group-1",
			}
			pods, err := k8sClient.K8sClientset.CoreV1().Pods("multiubuntu").List(context.TODO(), podSelector)
			Expect(err).To(BeNil())

			sout, _, err := util.K8sExecInPod(pods.Items[0].Name, "multiubuntu", []string{"cat", " /etc/hostname"})
			Expect(err).To(BeNil())
			fmt.Printf("OUTPUT: %s\n", sout)

			namespace := "default" // Modify if your deployment is in another namespace

			podList, err := k8sClient.K8sClientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
				LabelSelector: "app=syslog-server",
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(len(podList.Items)).To(BeNumerically(">", 0), "No pods found with label app=syslog-server")

			logRequest := k8sClient.K8sClientset.CoreV1().Pods(namespace).GetLogs(podList.Items[0].Name, &v1.PodLogOptions{})
			logs, err := logRequest.DoRaw(context.TODO())
			Expect(err).NotTo(HaveOccurred())

			fmt.Println(logs)

			Expect(string(logs)).To(ContainSubstring("CEF:0"))
		})
	})

})
