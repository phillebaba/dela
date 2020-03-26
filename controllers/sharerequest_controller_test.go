package controllers

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	sharev1alpha1 "github.com/phillebaba/dela/api/v1alpha1"
)

var _ = Describe("Share Request Controller", func() {
	const timeout = time.Second * 30
	const interval = time.Second * 1

	Context("New Cluster", func() {
		ctx := context.TODO()

		It("Should create a copy resource", func() {
			sourceNS := "ns1"
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "main",
					Namespace: sourceNS,
				},
				Data: map[string][]byte{"foo": []byte("bar")},
			}
			Expect(k8sClient.Create(ctx, secret)).Should(Succeed())
			shareIntent := &sharev1alpha1.ShareIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "main",
					Namespace: sourceNS,
				},
				Spec: sharev1alpha1.ShareIntentSpec{
					SecretReference: secret.Name,
				},
			}
			Expect(k8sClient.Create(ctx, shareIntent)).Should(Succeed())

			destNS := "ns2"
			shareRequest := &sharev1alpha1.ShareRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "main",
					Namespace: destNS,
				},
				Spec: sharev1alpha1.ShareRequestSpec{
					IntentReference: sharev1alpha1.ShareIntentReference{
						Name:      shareIntent.Name,
						Namespace: shareIntent.Namespace,
					},
				},
			}
			Expect(k8sClient.Create(ctx, shareRequest)).Should(Succeed())

			By("Expecting secret to be copied")
			Eventually(func() *corev1.Secret {
				secretCopy := &corev1.Secret{}
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: destNS}, secretCopy)
				return secretCopy
			}, timeout, interval).Should(SatisfyAll(
				WithTransform(func(e *corev1.Secret) string { return e.Name }, Equal(secret.Name)),
				WithTransform(func(e *corev1.Secret) int { return len(e.Data) }, Equal(len(secret.Data))),
				WithTransform(func(e *corev1.Secret) []byte { return e.Data["foo"] }, Equal(secret.Data["foo"])),
			))

			By("Expecting copied secret to be updated")
			secret.Data["foo"] = []byte("baz")
			Expect(k8sClient.Update(ctx, secret)).Should(Succeed())
			Eventually(func() *corev1.Secret {
				secretCopy := &corev1.Secret{}
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: destNS}, secretCopy)
				return secretCopy
			}, timeout, interval).Should(SatisfyAll(
				WithTransform(func(e *corev1.Secret) string { return e.Name }, Equal(secret.Name)),
				WithTransform(func(e *corev1.Secret) int { return len(e.Data) }, Equal(len(secret.Data))),
				WithTransform(func(e *corev1.Secret) []byte { return e.Data["foo"] }, Equal(secret.Data["foo"])),
			))

			/*By("Expecting to delete successfully")
			Eventually(func() error {
				f := &corev1alpha1.GeneratedSecret{}
				k8sClient.Get(ctx, key, f)
				return k8sClient.Delete(ctx, f)
			}, timeout, interval).Should(Succeed())
			Eventually(func() error {
				f := &corev1alpha1.GeneratedSecret{}
				return k8sClient.Get(ctx, key, f)
			}, timeout, interval).ShouldNot(Succeed())*/
		})
	})
})
