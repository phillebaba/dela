package controllers

import (
	"context"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	delav1alpha1 "github.com/phillebaba/dela/pkg/api/v1alpha1"
)

// IntentReconciler reconciles a Intent object
type IntentReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=dela.phillebaba.io,resources=intents,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=dela.phillebaba.io,resources=intents/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
func (r *IntentReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("intent", req.NamespacedName)

	intent := &delav1alpha1.Intent{}
	if err := r.Get(ctx, req.NamespacedName, intent); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	defer func() {
		if err := r.Status().Update(ctx, intent); err != nil {
			log.Error(err, "Could not update status")
		}
	}()

	secret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Name: intent.Spec.SecretName, Namespace: intent.Namespace}, secret); err != nil {
		intent.Status.State = delav1alpha1.IntentStateError
		r.Recorder.Event(intent, corev1.EventTypeNormal, "MissingSecret", "Can't get Secret specified by Intent")
		return ctrl.Result{}, err
	}

	if err := r.setOwnerReference(intent, secret); err != nil {
		intent.Status.State = delav1alpha1.IntentStateError
		r.Recorder.Event(intent, corev1.EventTypeNormal, "OwnerReference", "Could not set owner reference on Secret")
		return ctrl.Result{}, err
	}

	intent.Status.State = delav1alpha1.IntentStateReady
	return ctrl.Result{}, nil
}

func (r *IntentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&delav1alpha1.Intent{}).
		Watches(
			&source.Kind{Type: &corev1.Secret{}},
			&handler.EnqueueRequestForOwner{
				OwnerType:    &delav1alpha1.Intent{},
				IsController: false,
			},
		).
		Complete(r)
}

func (r *IntentReconciler) setOwnerReference(intent *delav1alpha1.Intent, secret *corev1.Secret) error {
	ctx := context.Background()
	if err := controllerutil.SetOwnerReference(intent, secret, r.Scheme); err != nil {
		return err
	}
	if err := r.Update(ctx, secret); err != nil {
		return err
	}

	return nil
}
