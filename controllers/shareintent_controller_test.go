package controllers

/*import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	sharev1alpha1 "github.com/phillebaba/dela/api/v1alpha1"
)*/

/*var _ = Describe("Share Intent Controller", func() {
	const timeout = time.Second * 30
	const interval = time.Second * 1

	Context("New Cluster", func() {
		ctx := context.TODO()
		ns := SetupTest(ctx)

		It("Should update status", func() {
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
			shareIntent := &sharev1alpha1.ShareIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: key.Namespace,
				},
				Spec: sharev1alpha1.ShareIntentSpec{
					SecretReference: secret.Name,
				},
			}

			By("Expecting status to be Ready")
			Expect(k8sClient.Create(ctx, secret)).Should(Succeed())
			Expect(k8sClient.Create(ctx, shareIntent)).Should(Succeed())
			Eventually(func() *sharev1alpha1.ShareIntent {
				shareIntent = &sharev1alpha1.ShareIntent{}
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: key.Name, Namespace: key.Namespace}, shareIntent)
				return shareIntent
			}, timeout, interval).Should(SatisfyAll(
				WithTransform(func(e *sharev1alpha1.ShareIntent) sharev1alpha1.ShareIntentState { return e.Status.State }, Equal(sharev1alpha1.SIReady)),
			))

			By("Expecting status to be Not Found")
			Expect(k8sClient.Delete(ctx, secret)).Should(Succeed())
			Eventually(func() *sharev1alpha1.ShareIntent {
				shareIntent = &sharev1alpha1.ShareIntent{}
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: key.Name, Namespace: key.Namespace}, shareIntent)
				return shareIntent
			}, timeout, interval).Should(SatisfyAll(
				WithTransform(func(e *sharev1alpha1.ShareIntent) sharev1alpha1.ShareIntentState { return e.Status.State }, Equal(sharev1alpha1.SINotFound)),
			))

			By("Expecting status to be Ready")
			secret.ObjectMeta.ResourceVersion = ""
			Expect(k8sClient.Create(ctx, secret)).Should(Succeed())
			Eventually(func() *sharev1alpha1.ShareIntent {
				shareIntent = &sharev1alpha1.ShareIntent{}
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: key.Name, Namespace: key.Namespace}, shareIntent)
				return shareIntent
			}, timeout, interval).Should(SatisfyAll(
				WithTransform(func(e *sharev1alpha1.ShareIntent) sharev1alpha1.ShareIntentState { return e.Status.State }, Equal(sharev1alpha1.SIReady)),
			))
		})
	})
})*/
