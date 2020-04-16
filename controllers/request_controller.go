package controllers

import (
	"context"
	"errors"
	"regexp"

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

	// Get reconciled Request
	request := &delav1alpha1.Request{}
	if err := r.Get(ctx, req.NamespacedName, request); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Function to update the Status before return
	defer func() {
		if err := r.Status().Update(ctx, request); err != nil {
			log.Error(err, "Could not update Request status", "status", request.Status)
		}
	}()

	// Get Intent for Request
	intentNN := types.NamespacedName{Name: request.Spec.IntentRef.Name, Namespace: request.Spec.IntentRef.Namespace}
	intent := &delav1alpha1.Intent{}
	if err := r.Get(ctx, intentNN, intent); err != nil {
		if apierrors.IsNotFound(err) {
			request.Status.State = delav1alpha1.RequestStateError
			log.Error(err, "Could not find Intent", "Request", req.NamespacedName, "Intent", intentNN)
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	}
	if intent.Status.State != delav1alpha1.IntentStateReady {
		request.Status.State = delav1alpha1.RequestStateError
		return ctrl.Result{}, nil
	}

	// Check if Request from namespace is whitelisted
	matches, err := matchesNamespaceWhitelist(request.Namespace, intent.Spec.NamespaceWhitelist)
	if err != nil {
		return ctrl.Result{}, err
	}
	if matches == false {
		log.Info("Intent does not allow Request from the namespace", "Namespace", req.Namespace, "Intent", intentNN, "Request", req.NamespacedName)
		request.Status.State = delav1alpha1.RequestStateError
		return ctrl.Result{}, nil
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
			request.Status.State = delav1alpha1.RequestStateError
			log.Info("Destination Secret already exists", "Secret", existSecret.Name)
			return ctrl.Result{}, errors.New("Destination alreay exists")
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
	var secretObjectMeta metav1.ObjectMeta
	if request.Spec.SecretConfig.ObjectMeta != nil {
		secretObjectMeta = *request.Spec.SecretConfig.ObjectMeta.DeepCopy()
		secretObjectMeta.Namespace = request.Namespace
	} else {
		secretObjectMeta = metav1.ObjectMeta{Name: secret.Name, Namespace: request.Namespace}
	}

	secretCopy := &corev1.Secret{ObjectMeta: secretObjectMeta}
	if _, err := ctrl.CreateOrUpdate(ctx, r, secretCopy, func() error {
		secretCopy.Data = secret.Data
		err := controllerutil.SetControllerReference(request, secretCopy, r.Scheme)
		return err
	}); err != nil {
		return ctrl.Result{}, err
	}

	r.Log.Info("Created or Updated Secret", "Secret", secret.Namespace+"/"+secret.Name)
	request.Status.State = delav1alpha1.RequestStateReady
	return ctrl.Result{}, nil
}

func (r *RequestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(&delav1alpha1.Request{}, ".metadata.intentRef", func(rawObj runtime.Object) []string {
		request := rawObj.(*delav1alpha1.Request)
		return []string{request.Spec.IntentRef.Namespace + "/" + request.Spec.IntentRef.Name}
	}); err != nil {
		return err
	}

	secretMapFn := handler.ToRequestsFunc(
		func(a handler.MapObject) []reconcile.Request {
			ctx := context.Background()

			var intents delav1alpha1.IntentList
			if err := r.List(ctx, &intents, client.InNamespace(a.Meta.GetNamespace()), client.MatchingField(".metadata.secretRef", a.Meta.GetName())); err != nil {
				return []reconcile.Request{}
			}

			reconcileReq := []reconcile.Request{}
			for _, intent := range intents.Items {
				var requests delav1alpha1.RequestList
				if err := r.List(ctx, &requests, client.MatchingField(".metadata.intentRef", intent.Namespace+"/"+intent.Name)); err != nil {
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

	intentMapFn := handler.ToRequestsFunc(
		func(a handler.MapObject) []reconcile.Request {
			ctx := context.Background()

			var requests delav1alpha1.RequestList
			if err := r.List(ctx, &requests, client.MatchingField(".metadata.intentRef", a.Meta.GetNamespace()+"/"+a.Meta.GetName())); err != nil {
				return []reconcile.Request{}
			}

			reconcileReq := []reconcile.Request{}
			for _, request := range requests.Items {
				reconcileReq = append(reconcileReq, reconcile.Request{NamespacedName: types.NamespacedName{
					Name:      request.Name,
					Namespace: request.Namespace,
				}})
			}

			return reconcileReq
		},
	)

	return ctrl.NewControllerManagedBy(mgr).
		For(&delav1alpha1.Request{}).
		Owns(&corev1.Secret{}).
		Watches(
			&source.Kind{Type: &corev1.Secret{}},
			&handler.EnqueueRequestsFromMapFunc{ToRequests: secretMapFn},
		).
		Watches(
			&source.Kind{Type: &delav1alpha1.Intent{}},
			&handler.EnqueueRequestsFromMapFunc{ToRequests: intentMapFn},
		).
		Complete(r)
}

// matchesNamespaceWhitelist checks if a given namespace matches the regex of any of the namespace whitelists
func matchesNamespaceWhitelist(namespace string, namespaceWhitelist []string) (bool, error) {
	if len(namespaceWhitelist) == 0 {
		return true, nil
	}

	for _, ns := range namespaceWhitelist {
		r, err := regexp.Compile(ns)
		if err != nil {
			return false, err
		}

		if r.MatchString(namespace) {
			return true, nil
		}
	}

	return false, nil
}
