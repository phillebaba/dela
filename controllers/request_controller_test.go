package controllers

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	delav1alpha1 "github.com/phillebaba/dela/api/v1alpha1"
)

var _ = Describe(" Request Controller", func() {
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
			intent := &delav1alpha1.Intent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "main",
					Namespace: source.Name,
				},
				Spec: delav1alpha1.IntentSpec{
					SecretReference: secret.Name,
				},
			}
			request := &delav1alpha1.Request{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "main",
					Namespace: dest.Name,
				},
				Spec: delav1alpha1.RequestSpec{
					IntentReference: delav1alpha1.IntentReference{
						Name:      intent.Name,
						Namespace: intent.Namespace,
					},
				},
			}

			By("Expecting secret to be copied")
			Expect(k8sClient.Create(ctx, secret)).Should(Succeed())
			Expect(k8sClient.Create(ctx, intent)).Should(Succeed())
			Expect(k8sClient.Create(ctx, request)).Should(Succeed())
			Eventually(func() *corev1.Secret {
				secretCopy := &corev1.Secret{}
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: dest.Name}, secretCopy)
				return secretCopy
			}, timeout, interval).Should(SatisfyAll(
				WithTransform(func(e *corev1.Secret) string { return e.Name }, Equal(secret.Name)),
				WithTransform(func(e *corev1.Secret) int { return len(e.Data) }, Equal(len(secret.Data))),
				WithTransform(func(e *corev1.Secret) []byte { return e.Data["foo"] }, Equal(secret.Data["foo"])),
			))
			Eventually(func() *delav1alpha1.Request {
				sr := &delav1alpha1.Request{}
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: request.Name, Namespace: request.Namespace}, sr)
				return sr
			}, timeout, interval).Should(SatisfyAll(
				WithTransform(func(e *delav1alpha1.Request) delav1alpha1.RequestState { return e.Status.State }, Equal(delav1alpha1.RReady)),
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
			intent := &delav1alpha1.Intent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "main",
					Namespace: source.Name,
				},
				Spec: delav1alpha1.IntentSpec{
					SecretReference: secret.Name,
				},
			}
			request := &delav1alpha1.Request{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "main",
					Namespace: dest.Name,
				},
				Spec: delav1alpha1.RequestSpec{
					IntentReference: delav1alpha1.IntentReference{
						Name:      intent.Name,
						Namespace: intent.Namespace,
					},
				},
			}

			By("Expecting Secret conflict")
			Expect(k8sClient.Create(ctx, secret)).Should(Succeed())
			Expect(k8sClient.Create(ctx, intent)).Should(Succeed())
			Expect(k8sClient.Create(ctx, request)).Should(Succeed())
			Eventually(func() *delav1alpha1.Request {
				sr := &delav1alpha1.Request{}
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: request.Name, Namespace: request.Namespace}, sr)
				return sr
			}, timeout, interval).Should(SatisfyAll(
				WithTransform(func(e *delav1alpha1.Request) delav1alpha1.RequestState { return e.Status.State }, Equal(delav1alpha1.RAlreadyExists)),
			))

			By("Expecting Intent to not be found")
			Expect(k8sClient.Delete(ctx, intent)).Should(Succeed())
			Eventually(func() *delav1alpha1.Request {
				sr := &delav1alpha1.Request{}
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: request.Name, Namespace: request.Namespace}, sr)
				return sr
			}, timeout, interval).Should(SatisfyAll(
				WithTransform(func(e *delav1alpha1.Request) delav1alpha1.RequestState { return e.Status.State }, Equal(delav1alpha1.RNotFound)),
			))
		})
	})
})
