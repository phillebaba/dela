package controllers

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	delav1alpha1 "github.com/phillebaba/dela/pkg/api/v1alpha1"
)

var _ = Describe("Intent Controller", func() {
	const timeout = time.Second * 30
	const interval = time.Second * 1

	ctx := context.TODO()
	ns := SetupTestNamespace(ctx)

	Context("New Cluster", func() {
		It("Should update the Intent status", func() {
			key := types.NamespacedName{
				Name:      "main",
				Namespace: ns.Name,
			}
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: key.Namespace,
				},
			}
			intent := &delav1alpha1.Intent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: key.Namespace,
				},
				Spec: delav1alpha1.IntentSpec{
					SecretName: secret.Name,
				},
			}

			By("Creating a Secret and Intent")
			Expect(k8sClient.Create(ctx, secret)).Should(Succeed())
			Expect(k8sClient.Create(ctx, intent)).Should(Succeed())
			Eventually(func() *delav1alpha1.Intent {
				intent = &delav1alpha1.Intent{}
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: key.Name, Namespace: key.Namespace}, intent)
				return intent
			}, timeout, interval).Should(SatisfyAll(
				WithTransform(func(e *delav1alpha1.Intent) delav1alpha1.IntentState { return e.Status.State }, Equal(delav1alpha1.IntentStateReady)),
			))

			By("Deleting the Secret")
			Expect(k8sClient.Delete(ctx, secret)).Should(Succeed())
			Eventually(func() *delav1alpha1.Intent {
				intent = &delav1alpha1.Intent{}
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: key.Name, Namespace: key.Namespace}, intent)
				return intent
			}, timeout, interval).Should(SatisfyAll(
				WithTransform(func(e *delav1alpha1.Intent) delav1alpha1.IntentState { return e.Status.State }, Equal(delav1alpha1.IntentStateError)),
			))

			By("Re-creating the Secret")
			secret.ObjectMeta.ResourceVersion = ""
			Expect(k8sClient.Create(ctx, secret)).Should(Succeed())
			Eventually(func() *delav1alpha1.Intent {
				intent = &delav1alpha1.Intent{}
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: key.Name, Namespace: key.Namespace}, intent)
				return intent
			}, timeout, interval).Should(SatisfyAll(
				WithTransform(func(e *delav1alpha1.Intent) delav1alpha1.IntentState { return e.Status.State }, Equal(delav1alpha1.IntentStateError)),
			))
		})
	})
})
