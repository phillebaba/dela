package controllers

import (
	"context"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	sharev1alpha1 "github.com/phillebaba/dela/api/v1alpha1"
)

// ShareRequestReconciler reconciles a ShareRequest object
type ShareRequestReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=share.phillebaba.io,resources=sharerequests,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=share.phillebaba.io,resources=sharerequests/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete

func (r *ShareRequestReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("sharerequest", req.NamespacedName)

	shareRequest := &sharev1alpha1.ShareRequest{}
	if err := r.Get(ctx, req.NamespacedName, shareRequest); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	shareIntentNN := types.NamespacedName{Name: shareRequest.Spec.IntentReference.Name, Namespace: shareRequest.Spec.IntentReference.Namespace}
	shareIntent := &sharev1alpha1.ShareIntent{}
	if err := r.Get(ctx, shareIntentNN, shareIntent); err != nil {
		log.Error(err, "Could not get ShareRequests referenced ShareIntent", "ShareRequest", req.NamespacedName, "ShareIntent", shareIntentNN)
		return ctrl.Result{}, err
	}

	secretNN := types.NamespacedName{Name: shareIntent.Spec.SecretReference, Namespace: shareIntentNN.Namespace}
	secret := &corev1.Secret{}
	if err := r.Get(ctx, secretNN, secret); err != nil {
		log.Error(err, "Could not get ShareIntents referenced Secret", "ShareIntent", shareIntentNN, "Secret", secretNN)
		return ctrl.Result{}, err
	}

	secretCopy := secret.DeepCopy()
	secretCopy.ResourceVersion = ""
	secretCopy.ObjectMeta.Namespace = shareRequest.Namespace
	_, err := ctrl.CreateOrUpdate(ctx, r, secretCopy, func() error {
		return controllerutil.SetControllerReference(shareRequest, secretCopy, r.Scheme)
	})

	if err != nil {
		return ctrl.Result{}, err
	}

	r.Log.Info("Created or Updated Secret", "Secret", secret.Namespace+"/"+secret.Name)
	return ctrl.Result{}, nil
}

func (r *ShareRequestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&sharev1alpha1.ShareRequest{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}
