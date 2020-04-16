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

	delav1alpha1 "github.com/phillebaba/dela/api/v1alpha1"
)

// IntentReconciler reconciles a Intent object
type IntentReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=dela.phillebaba.io,resources=intents,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=dela.phillebaba.io,resources=intents/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
func (r *IntentReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("intent", req.NamespacedName)

	intent := &delav1alpha1.Intent{}
	if err := r.Get(ctx, req.NamespacedName, intent); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	defer func() {
		if err := r.Status().Update(ctx, intent); err != nil {
			log.Error(err, "Could not update Intent status")
		}
	}()

	secret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Name: intent.Spec.SecretReference, Namespace: intent.Namespace}, secret); err != nil {
		log.Error(err, "Could not get Secret refered by Intent", "Intent", intent.Name, "Secret", intent.Spec.SecretReference)
		intent.Status.State = delav1alpha1.IntentStateError
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	intent.Status.State = delav1alpha1.IntentStateReady
	return ctrl.Result{}, nil
}

func (r *IntentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(&delav1alpha1.Intent{}, ".metadata.secretRef", func(rawObj runtime.Object) []string {
		intent := rawObj.(*delav1alpha1.Intent)
		return []string{intent.Spec.SecretReference}
	}); err != nil {
		return err
	}

	mapFn := handler.ToRequestsFunc(
		func(a handler.MapObject) []reconcile.Request {
			ctx := context.Background()

			var intents delav1alpha1.IntentList
			if err := r.List(ctx, &intents, client.InNamespace(a.Meta.GetNamespace()), client.MatchingField(".metadata.secretRef", a.Meta.GetName())); err != nil {
				return []reconcile.Request{}
			}

			requests := []reconcile.Request{}
			for _, intent := range intents.Items {
				requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{
					Name:      intent.Name,
					Namespace: intent.Namespace,
				}})
			}

			return requests
		},
	)

	return ctrl.NewControllerManagedBy(mgr).
		For(&delav1alpha1.Intent{}).
		Watches(
			&source.Kind{Type: &corev1.Secret{}},
			&handler.EnqueueRequestsFromMapFunc{ToRequests: mapFn},
		).
		Complete(r)
}
