package controllers

import (
	"context"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

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

		shareIntent.Status.State = sharev1alpha1.SecretNotFound
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
	return ctrl.NewControllerManagedBy(mgr).
		For(&sharev1alpha1.ShareIntent{}).
		Complete(r)
}
