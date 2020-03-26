package controllers

import (
	"context"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	sharev1alpha1 "github.com/phillebaba/dela/api/v1alpha1"
)

// ShareIntentReconciler reconciles a ShareIntent object
type ShareIntentReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=share.phillebaba.io,resources=shareintents,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=share.phillebaba.io,resources=shareintents/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete

func (r *ShareIntentReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("shareintent", req.NamespacedName)

	shareIntent := &sharev1alpha1.ShareIntent{}
	if err := r.Get(ctx, req.NamespacedName, shareIntent); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	secret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Name: shareIntent.Spec.SecretReference, Namespace: shareIntent.Namespace}, secret); err != nil {
		log.Error(err, "Could not get ShareIntents referenced Secret", "ShareIntent", shareIntent.Name, "Secret", shareIntent.Spec.SecretReference)

		shareIntent.Status.State = sharev1alpha1.NotFound
		err := r.Status().Update(ctx, shareIntent)
		if err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, err
	}

	shareIntent.Status.State = sharev1alpha1.Ready
	err := r.Status().Update(ctx, shareIntent)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ShareIntentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(&sharev1alpha1.ShareIntent{}, ".metadata.secretRef", func(rawObj runtime.Object) []string {
		shareIntent := rawObj.(*sharev1alpha1.ShareIntent)
		return []string{shareIntent.Spec.SecretReference}
	}); err != nil {
		return err
	}

	mapFn := handler.ToRequestsFunc(
		func(a handler.MapObject) []reconcile.Request {
			ctx := context.Background()

			var shareIntents sharev1alpha1.ShareIntentList
			if err := r.List(ctx, &shareIntents, client.InNamespace(a.Meta.GetNamespace()), client.MatchingField(".metadata.secretRef", a.Meta.GetName())); err != nil {
				return nil
			}

			requests := []reconcile.Request{}
			for _, shareIntent := range shareIntents.Items {
				requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{
					Name:      shareIntent.Name,
					Namespace: shareIntent.Namespace,
				}})
			}

			return requests
		},
	)

	return ctrl.NewControllerManagedBy(mgr).
		For(&sharev1alpha1.ShareIntent{}).
		Watches(
			&source.Kind{Type: &corev1.Secret{}},
			&handler.EnqueueRequestsFromMapFunc{ToRequests: mapFn},
		).
		Complete(r)
}
