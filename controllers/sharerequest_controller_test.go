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
		source := SetupTestNamespace(ctx)
		dest := SetupTestNamespace(ctx)

		It("Should create a copy resource", func() {
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "main",
					Namespace: source.Name,
				},
				Data: map[string][]byte{"foo": []byte("bar")},
			}
			shareIntent := &sharev1alpha1.ShareIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "main",
					Namespace: source.Name,
				},
				Spec: sharev1alpha1.ShareIntentSpec{
					SecretReference: secret.Name,
				},
			}
			shareRequest := &sharev1alpha1.ShareRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "main",
					Namespace: dest.Name,
				},
				Spec: sharev1alpha1.ShareRequestSpec{
					IntentReference: sharev1alpha1.ShareIntentReference{
						Name:      shareIntent.Name,
						Namespace: shareIntent.Namespace,
					},
				},
			}

			By("Expecting secret to be copied")
			Expect(k8sClient.Create(ctx, secret)).Should(Succeed())
			Expect(k8sClient.Create(ctx, shareIntent)).Should(Succeed())
			Expect(k8sClient.Create(ctx, shareRequest)).Should(Succeed())
			Eventually(func() *corev1.Secret {
				secretCopy := &corev1.Secret{}
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: dest.Name}, secretCopy)
				return secretCopy
			}, timeout, interval).Should(SatisfyAll(
				WithTransform(func(e *corev1.Secret) string { return e.Name }, Equal(secret.Name)),
				WithTransform(func(e *corev1.Secret) int { return len(e.Data) }, Equal(len(secret.Data))),
				WithTransform(func(e *corev1.Secret) []byte { return e.Data["foo"] }, Equal(secret.Data["foo"])),
			))
			Eventually(func() *sharev1alpha1.ShareRequest {
				sr := &sharev1alpha1.ShareRequest{}
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: shareRequest.Name, Namespace: shareRequest.Namespace}, sr)
				return sr
			}, timeout, interval).Should(SatisfyAll(
				WithTransform(func(e *sharev1alpha1.ShareRequest) sharev1alpha1.ShareRequestState { return e.Status.State }, Equal(sharev1alpha1.SRReady)),
			))

			By("Expecting copied secret to be updated")
			secret.Data["foo"] = []byte("baz")
			Expect(k8sClient.Update(ctx, secret)).Should(Succeed())
			Eventually(func() *corev1.Secret {
				secretCopy := &corev1.Secret{}
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: dest.Name}, secretCopy)
				return secretCopy
			}, timeout, interval).Should(SatisfyAll(
				WithTransform(func(e *corev1.Secret) string { return e.Name }, Equal(secret.Name)),
				WithTransform(func(e *corev1.Secret) int { return len(e.Data) }, Equal(len(secret.Data))),
				WithTransform(func(e *corev1.Secret) []byte { return e.Data["foo"] }, Equal(secret.Data["foo"])),
			))
		})
	})

	Context("Cluster with existing secret", func() {
		ctx := context.TODO()
		source := SetupTestNamespace(ctx)
		dest := SetupTestNamespace(ctx)

		It("Should update status", func() {
			existSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "main",
					Namespace: dest.Name,
				},
			}
			Expect(k8sClient.Create(ctx, existSecret)).Should(Succeed())

			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      existSecret.Name,
					Namespace: source.Name,
				},
			}
			shareIntent := &sharev1alpha1.ShareIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "main",
					Namespace: source.Name,
				},
				Spec: sharev1alpha1.ShareIntentSpec{
					SecretReference: secret.Name,
				},
			}
			shareRequest := &sharev1alpha1.ShareRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "main",
					Namespace: dest.Name,
				},
				Spec: sharev1alpha1.ShareRequestSpec{
					IntentReference: sharev1alpha1.ShareIntentReference{
						Name:      shareIntent.Name,
						Namespace: shareIntent.Namespace,
					},
				},
			}

			By("Expecting Secret conflict")
			Expect(k8sClient.Create(ctx, secret)).Should(Succeed())
			Expect(k8sClient.Create(ctx, shareIntent)).Should(Succeed())
			Expect(k8sClient.Create(ctx, shareRequest)).Should(Succeed())
			Eventually(func() *sharev1alpha1.ShareRequest {
				sr := &sharev1alpha1.ShareRequest{}
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: shareRequest.Name, Namespace: shareRequest.Namespace}, sr)
				return sr
			}, timeout, interval).Should(SatisfyAll(
				WithTransform(func(e *sharev1alpha1.ShareRequest) sharev1alpha1.ShareRequestState { return e.Status.State }, Equal(sharev1alpha1.SRAlreadyExists)),
			))

			By("Expecting ShareIntent to not be found")
			Expect(k8sClient.Delete(ctx, shareIntent)).Should(Succeed())
			Eventually(func() *sharev1alpha1.ShareRequest {
				sr := &sharev1alpha1.ShareRequest{}
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: shareRequest.Name, Namespace: shareRequest.Namespace}, sr)
				return sr
			}, timeout, interval).Should(SatisfyAll(
				WithTransform(func(e *sharev1alpha1.ShareRequest) sharev1alpha1.ShareRequestState { return e.Status.State }, Equal(sharev1alpha1.SRNotFound)),
			))
		})
	})
})
