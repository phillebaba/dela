package controllers

import (
	"context"
	"errors"
	"regexp"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	delav1alpha1 "github.com/phillebaba/dela/api/v1alpha1"
)

// RequestReconciler reconciles a Request object
type RequestReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=delete.phillebaba.io,resources=requests,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=dela.phillebaba.io,resources=requests/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
func (r *RequestReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("request", req.NamespacedName)

	request := &delav1alpha1.Request{}
	if err := r.Get(ctx, req.NamespacedName, request); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	defer func() {
		if err := r.Status().Update(ctx, request); err != nil {
			log.Error(err, "Could not update Request status")
		}
	}()

	// Get Intent for Request
	intentNN := types.NamespacedName{Name: request.Spec.IntentReference.Name, Namespace: request.Spec.IntentReference.Namespace}
	intent := &delav1alpha1.Intent{}
	if err := r.Get(ctx, intentNN, intent); err != nil {
		if apierrors.IsNotFound(err) {
			request.Status.State = delav1alpha1.RNotFound
		}

		log.Error(err, "Could not get Intent referenced Request", "Request", req.NamespacedName, "Intent", intentNN)
		return ctrl.Result{}, err
	}

	// Check if Request from namespace is allowed
	matches, err := matchesAllowedNamespace(request.Namespace, intent.Spec.AllowedNamespaces)
	if err != nil {
		return ctrl.Result{}, err
	}

	if matches == false {
		log.Info("Intent does not allow Request from the current namespace", "Namespace", req.Namespace, "Intent", intentNN, "Request", req.NamespacedName)
		return ctrl.Result{Requeue: true, RequeueAfter: time.Minute * 1}, nil
	}

	// Make sure Secret destination does not already exist
	existSecret := &corev1.Secret{}
	err = r.Get(ctx, types.NamespacedName{Name: intent.Spec.SecretReference, Namespace: req.Namespace}, existSecret)
	if client.IgnoreNotFound(err) != nil {
		return ctrl.Result{}, err
	}
	if err == nil && existSecret != nil {
		owner := metav1.GetControllerOf(existSecret)
		if owner == nil || owner.Kind != "Request" && owner.Name != request.Name {
			request.Status.State = delav1alpha1.RAlreadyExists
			log.Info("Destination Secret already exists", "Secret", existSecret.Name)
			return ctrl.Result{}, errors.New("Secret already exists")
		}
	}

	// Get Secret for Intent
	secretNN := types.NamespacedName{Name: intent.Spec.SecretReference, Namespace: intentNN.Namespace}
	secret := &corev1.Secret{}
	if err := r.Get(ctx, secretNN, secret); err != nil {
		log.Error(err, "Could not get Intents referenced Secret", "Intent", intentNN, "Secret", secretNN)
		return ctrl.Result{}, err
	}

	// Create copy of Intents Secret
	secretCopy := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: secret.Name, Namespace: req.Namespace}}
	_, err = ctrl.CreateOrUpdate(ctx, r, secretCopy, func() error {
		secretCopy.Annotations = map[string]string{}
		secretCopy.Annotations["phillebaba.io/generated-from"] = request.Namespace + "/" + request.Name
		secretCopy.Data = secret.Data
		return controllerutil.SetControllerReference(request, secretCopy, r.Scheme)
	})

	if err != nil {
		return ctrl.Result{}, err
	}

	request.Status.State = delav1alpha1.RReady
	r.Log.Info("Created or Updated Secret", "Secret", secret.Namespace+"/"+secret.Name)
	return ctrl.Result{}, nil
}

func (r *RequestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(&delav1alpha1.Request{}, ".metadata.intentRef", func(rawObj runtime.Object) []string {
		request := rawObj.(*delav1alpha1.Request)
		return []string{request.Spec.IntentReference.Name + "/" + request.Spec.IntentReference.Namespace}
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

			reconcileReq := []reconcile.Request{}
			for _, intent := range intents.Items {
				var requests delav1alpha1.RequestList
				if err := r.List(ctx, &requests, client.MatchingField(".metadata.intentRef", intent.Name+"/"+intent.Namespace)); err != nil {
					return []reconcile.Request{}
				}

				for _, request := range requests.Items {
					reconcileReq = append(reconcileReq, reconcile.Request{NamespacedName: types.NamespacedName{
						Name:      request.Name,
						Namespace: request.Namespace,
					}})
				}
			}

			return reconcileReq
		},
	)

	return ctrl.NewControllerManagedBy(mgr).
		For(&delav1alpha1.Request{}).
		Owns(&corev1.Secret{}).
		Watches(
			&source.Kind{Type: &corev1.Secret{}},
			&handler.EnqueueRequestsFromMapFunc{ToRequests: mapFn},
		).
		Complete(r)
}

// matchesAllowedNamespace checks if a given namespace matches the regex of any of the allowed namespaces
func matchesAllowedNamespace(namespace string, allowedNamespaces []string) (bool, error) {
	if len(allowedNamespaces) == 0 {
		return true, nil
	}

	for _, allowedNamespace := range allowedNamespaces {
		r, err := regexp.Compile(allowedNamespace)
		if err != nil {
			return false, err
		}

		if r.MatchString(namespace) {
			return true, nil
		}
	}

	return false, nil
}
