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

var _ = Describe("Request Controller", func() {
	const timeout = time.Second * 30
	const interval = time.Second * 1

	ctx := context.TODO()
	source := SetupTestNamespace(ctx)
	dest := SetupTestNamespace(ctx)

	Context("New Cluster", func() {
		It("Creates a copy of a Secret", func() {
			secret, intent, request := baseResources(source, dest)
			intent.Spec.NamespaceWhitelist = []string{dest.Name}

			By("Creating a Secret, Intent and Request")
			Expect(k8sClient.Create(ctx, secret)).Should(Succeed())
			Expect(k8sClient.Create(ctx, intent)).Should(Succeed())
			Expect(k8sClient.Create(ctx, request)).Should(Succeed())
			Eventually(func() *corev1.Secret {
				secretCopy := &corev1.Secret{}
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: request.Spec.SecretObjectMeta.Name, Namespace: request.Namespace}, secretCopy)
				return secretCopy
			}, timeout, interval).Should(SatisfyAll(
				WithTransform(func(e *corev1.Secret) int { return len(e.Data) }, Equal(len(secret.Data))),
				WithTransform(func(e *corev1.Secret) []byte { return e.Data["foo"] }, Equal(secret.Data["foo"])),
			))
			Eventually(func() *delav1alpha1.Request {
				sr := &delav1alpha1.Request{}
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: request.Name, Namespace: request.Namespace}, sr)
				return sr
			}, timeout, interval).Should(
				WithTransform(func(e *delav1alpha1.Request) delav1alpha1.RequestState { return e.Status.State }, Equal(delav1alpha1.RequestStateReady)),
			)

			By("Updating the Secret data")
			secret.Data["foo"] = []byte("baz")
			Expect(k8sClient.Update(ctx, secret)).Should(Succeed())
			Eventually(func() *corev1.Secret {
				secretCopy := &corev1.Secret{}
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: request.Spec.SecretObjectMeta.Name, Namespace: request.Namespace}, secretCopy)
				return secretCopy
			}, timeout, interval).Should(SatisfyAll(
				WithTransform(func(e *corev1.Secret) int { return len(e.Data) }, Equal(len(secret.Data))),
				WithTransform(func(e *corev1.Secret) []byte { return e.Data["foo"] }, Equal(secret.Data["foo"])),
			))
		})

		It("Triggers an update of a Request from an Intent change", func() {
			secret, intent, request := baseResources(source, dest)

			By("Creating a Request and Secret")
			Expect(k8sClient.Create(ctx, secret)).Should(Succeed())
			Expect(k8sClient.Create(ctx, request)).Should(Succeed())
			Eventually(func() *delav1alpha1.Request {
				r := &delav1alpha1.Request{}
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: request.Name, Namespace: request.Namespace}, r)
				return r
			}, timeout, interval).Should(
				WithTransform(func(e *delav1alpha1.Request) delav1alpha1.RequestState { return e.Status.State }, Equal(delav1alpha1.RequestStateError)),
			)

			By("Creating an Intent")
			Expect(k8sClient.Create(ctx, intent)).Should(Succeed())
			Eventually(func() *delav1alpha1.Request {
				r := &delav1alpha1.Request{}
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: request.Name, Namespace: request.Namespace}, r)
				return r
			}, timeout, interval).Should(
				WithTransform(func(e *delav1alpha1.Request) delav1alpha1.RequestState { return e.Status.State }, Equal(delav1alpha1.RequestStateReady)),
			)
		})

		It("Handles missing Secret for Intent", func() {
			_, intent, request := baseResources(source, dest)

			By("Creating a Intent and a Request")
			Expect(k8sClient.Create(ctx, intent)).Should(Succeed())
			Expect(k8sClient.Create(ctx, request)).Should(Succeed())
			Eventually(func() *delav1alpha1.Request {
				r := &delav1alpha1.Request{}
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: request.Name, Namespace: request.Namespace}, r)
				return r
			}, timeout, interval).Should(
				WithTransform(func(e *delav1alpha1.Request) delav1alpha1.RequestState { return e.Status.State }, Equal(delav1alpha1.RequestStateError)),
			)
		})

		It("Does not copy Secrets to Namespaces that are not whitelisted", func() {
			secret, intent, request := baseResources(source, dest)
			intent.Spec.NamespaceWhitelist = []string{dest.Name + "-extra"}

			By("Creating an Intent and Secret")
			Expect(k8sClient.Create(ctx, secret)).Should(Succeed())
			Expect(k8sClient.Create(ctx, intent)).Should(Succeed())

			By("Creating a Request in a non whitespaced Namespace")
			Expect(k8sClient.Create(ctx, request)).Should(Succeed())
			Eventually(func() *delav1alpha1.Request {
				r := &delav1alpha1.Request{}
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: request.Name, Namespace: request.Namespace}, r)
				return r
			}, timeout, interval).Should(
				WithTransform(func(e *delav1alpha1.Request) delav1alpha1.RequestState { return e.Status.State }, Equal(delav1alpha1.RequestStateError)),
			)
		})

		It("Does not delete copied Secret when the Intent is deleted", func() {
			secret, intent, request := baseResources(source, dest)

			By("Creating an Intent, Secret, and Request")
			Expect(k8sClient.Create(ctx, secret)).Should(Succeed())
			Expect(k8sClient.Create(ctx, intent)).Should(Succeed())
			Expect(k8sClient.Create(ctx, request)).Should(Succeed())
			Eventually(func() error {
				secretCopy := &corev1.Secret{}
				return k8sClient.Get(ctx, types.NamespacedName{Name: request.Spec.SecretObjectMeta.Name, Namespace: request.Namespace}, secretCopy)
			}).Should(Succeed())

			By("Deleting the Intent")
			Expect(k8sClient.Delete(ctx, intent)).Should(Succeed())
			Eventually(func() *delav1alpha1.Request {
				r := &delav1alpha1.Request{}
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: request.Name, Namespace: request.Namespace}, r)
				return r
			}, timeout, interval).Should(
				WithTransform(func(e *delav1alpha1.Request) delav1alpha1.RequestState { return e.Status.State }, Equal(delav1alpha1.RequestStateError)),
			)
			Eventually(func() error {
				secretCopy := &corev1.Secret{}
				return k8sClient.Get(ctx, types.NamespacedName{Name: request.Spec.SecretObjectMeta.Name, Namespace: request.Namespace}, secretCopy)
			}).Should(Succeed())
		})

		It("Does not delete copied Secret when the source Secret is deleted", func() {
			secret, intent, request := baseResources(source, dest)

			By("Creating an Intent, Secret, and Request")
			Expect(k8sClient.Create(ctx, secret)).Should(Succeed())
			Expect(k8sClient.Create(ctx, intent)).Should(Succeed())
			Expect(k8sClient.Create(ctx, request)).Should(Succeed())
			Eventually(func() error {
				secretCopy := &corev1.Secret{}
				return k8sClient.Get(ctx, types.NamespacedName{Name: request.Spec.SecretObjectMeta.Name, Namespace: request.Namespace}, secretCopy)
			}).Should(Succeed())

			By("Deleting the Secret")
			Expect(k8sClient.Delete(ctx, secret)).Should(Succeed())
			Eventually(func() *delav1alpha1.Request {
				r := &delav1alpha1.Request{}
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: request.Name, Namespace: request.Namespace}, r)
				return r
			}, timeout, interval).Should(
				WithTransform(func(e *delav1alpha1.Request) delav1alpha1.RequestState { return e.Status.State }, Equal(delav1alpha1.RequestStateError)),
			)
			Eventually(func() error {
				secretCopy := &corev1.Secret{}
				return k8sClient.Get(ctx, types.NamespacedName{Name: request.Spec.SecretObjectMeta.Name, Namespace: request.Namespace}, secretCopy)
			}).Should(Succeed())
		})

		It("Updates Secret copy when ObjectMeta changes", func() {
			secret, intent, request := baseResources(source, dest)

			By("Creating an Intent, Secret, and Request")
			Expect(k8sClient.Create(ctx, secret)).Should(Succeed())
			Expect(k8sClient.Create(ctx, intent)).Should(Succeed())
			Expect(k8sClient.Create(ctx, request)).Should(Succeed())
			Eventually(func() error {
				secretCopy := &corev1.Secret{}
				return k8sClient.Get(ctx, types.NamespacedName{Name: request.Spec.SecretObjectMeta.Name, Namespace: request.Namespace}, secretCopy)
			}).Should(Succeed())

			By("Changing the Secret copy name")
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: request.Name, Namespace: request.Namespace}, request)).Should(Succeed())
			oldName := request.Spec.SecretObjectMeta.Name
			request.Spec.SecretObjectMeta.Name = "something-new"
			Expect(k8sClient.Update(ctx, request)).Should(Succeed())
			Eventually(func() error {
				secretCopy := &corev1.Secret{}
				return k8sClient.Get(ctx, types.NamespacedName{Name: request.Spec.SecretObjectMeta.Name, Namespace: request.Namespace}, secretCopy)
			}).Should(Succeed())

			Eventually(func() error {
				secretCopy := &corev1.Secret{}
				return k8sClient.Get(ctx, types.NamespacedName{Name: oldName, Namespace: request.Namespace}, secretCopy)
			}).ShouldNot(Succeed())
		})
	})

	Context("Cluster with existing secret", func() {
		var existSecret *corev1.Secret
		BeforeEach(func() {
			existSecret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "main",
					Namespace: dest.Name,
				},
			}
			Expect(k8sClient.Create(ctx, existSecret)).Should(Succeed())
		})

		It("Detects the conflicting destination Secret", func() {
			secret, intent, request := baseResources(source, dest)
			request.Spec.SecretObjectMeta.Name = existSecret.Name

			By("Creating a source Secret, Intent, and Request")
			Expect(k8sClient.Create(ctx, secret)).Should(Succeed())
			Expect(k8sClient.Create(ctx, intent)).Should(Succeed())
			Expect(k8sClient.Create(ctx, request)).Should(Succeed())
			Eventually(func() *delav1alpha1.Request {
				sr := &delav1alpha1.Request{}
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: request.Name, Namespace: request.Namespace}, sr)
				return sr
			}, timeout, interval).Should(SatisfyAll(
				WithTransform(func(e *delav1alpha1.Request) delav1alpha1.RequestState { return e.Status.State }, Equal(delav1alpha1.RequestStateError)),
			))

			By("Deleting conflicting destination Secret")
			Expect(k8sClient.Delete(ctx, existSecret)).Should(Succeed())
			Eventually(func() *delav1alpha1.Request {
				sr := &delav1alpha1.Request{}
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: request.Name, Namespace: request.Namespace}, sr)
				return sr
			}, timeout, interval).Should(SatisfyAll(
				WithTransform(func(e *delav1alpha1.Request) delav1alpha1.RequestState { return e.Status.State }, Equal(delav1alpha1.RequestStateReady)),
			))
		})
	})
})

// Creates a base Secret, Intent, and Request for tests.
func baseResources(source *corev1.Namespace, dest *corev1.Namespace) (*corev1.Secret, *delav1alpha1.Intent, *delav1alpha1.Request) {
	name := "main"

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: source.Name,
		},
		Data: map[string][]byte{"foo": []byte("bar")},
	}
	intent := &delav1alpha1.Intent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: source.Name,
		},
		Spec: delav1alpha1.IntentSpec{
			SecretReference: secret.Name,
		},
	}
	request := &delav1alpha1.Request{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: dest.Name,
		},
		Spec: delav1alpha1.RequestSpec{
			IntentRef: delav1alpha1.IntentReference{
				Name:      intent.Name,
				Namespace: intent.Namespace,
			},
			SecretObjectMeta: metav1.ObjectMeta{
				Name:      name + "-copy",
				Namespace: dest.Name + "-wrong",
			},
		},
	}

	return secret, intent, request
}
