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
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	delav1alpha1 "github.com/phillebaba/dela/api/v1alpha1"
)

var (
	jobOwnerKey = ".metadata.controller"
	apiGVStr    = delav1alpha1.GroupVersion.String()
)

// RequestReconciler reconciles a Request object
type RequestReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=delete.phillebaba.io,resources=requests,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=dela.phillebaba.io,resources=requests/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
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
			log.Error(err, "Could not update status")
		}
	}()

	// Make sure Secret destination does not already exist
	existSecret := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{Name: request.Spec.SecretObjectMeta.Name, Namespace: request.Namespace}, existSecret)
	if client.IgnoreNotFound(err) != nil {
		request.Status.State = delav1alpha1.RequestStateError
		return ctrl.Result{}, err
	}
	if err == nil && existSecret != nil {
		owner := metav1.GetControllerOf(existSecret)
		if owner == nil || owner.Kind != "Request" && owner.Name != request.Name {
			request.Status.State = delav1alpha1.RequestStateError
			r.Recorder.Event(request, corev1.EventTypeNormal, "SecretExists", "Destination Secret already exists")
			return ctrl.Result{}, errors.New("Destination alreay exists")
		}
	}

	// Get Intent for Request
	intentNN := types.NamespacedName{Name: request.Spec.IntentRef.Name, Namespace: request.Spec.IntentRef.Namespace}
	intent := &delav1alpha1.Intent{}
	if err := r.Get(ctx, intentNN, intent); err != nil {
		if apierrors.IsNotFound(err) {
			request.Status.State = delav1alpha1.RequestStateError
			r.Recorder.Event(request, corev1.EventTypeNormal, "MissingIntent", "Could not find referenced Intent")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	if intent.Status.State != delav1alpha1.IntentStateReady {
		request.Status.State = delav1alpha1.RequestStateError
		r.Recorder.Event(request, corev1.EventTypeNormal, "IntentNotReady", "Intent not in ready state")
		return ctrl.Result{}, nil
	}

	// Check if Request from namespace is whitelisted
	matches, err := matchesNamespaceWhitelist(request.Namespace, intent.Spec.NamespaceWhitelist)
	if err != nil {
		request.Status.State = delav1alpha1.RequestStateError
		return ctrl.Result{}, err
	}
	if matches == false {
		request.Status.State = delav1alpha1.RequestStateError
		r.Recorder.Event(request, corev1.EventTypeNormal, "Forbidden", "Intent does not allow request from namespace")
		return ctrl.Result{}, nil
	}

	// Get Secret referenced by Intent
	secretNN := types.NamespacedName{Name: intent.Spec.SecretName, Namespace: intent.Namespace}
	secret := &corev1.Secret{}
	if err := r.Get(ctx, secretNN, secret); err != nil {
		return ctrl.Result{}, err
	}

	// Create Secret copy
	secretCopy := &corev1.Secret{ObjectMeta: request.Spec.SecretObjectMeta}
	secretCopy.ObjectMeta.Namespace = request.Namespace
	result, err := ctrl.CreateOrUpdate(ctx, r, secretCopy, func() error {
		secretCopy.Data = secret.Data
		err := controllerutil.SetControllerReference(request, secretCopy, r.Scheme)
		return err
	})
	if err != nil {
		request.Status.State = delav1alpha1.RequestStateError
		r.Recorder.Event(request, corev1.EventTypeNormal, "Failed", "Could not create Secret copy")
		return ctrl.Result{}, err
	}

	// Delete Secret copy if SecretObjectMeta has changed name
	var childSecrets corev1.SecretList
	if err := r.List(ctx, &childSecrets, client.InNamespace(req.Namespace), client.MatchingFields{jobOwnerKey: req.Name}); err != nil {
		return ctrl.Result{}, err
	}
	for _, childSecret := range childSecrets.Items {
		if childSecret.Name != request.Spec.SecretObjectMeta.Name {
			log.Info("Deleting old Secret copy due to name change", "old", childSecret.Name, "new", request.Spec.SecretObjectMeta.Name)
			if err := r.Delete(ctx, &childSecret); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	// Creation completed sucessfully
	request.Status.State = delav1alpha1.RequestStateReady
	if result == controllerutil.OperationResultCreated {
		r.Recorder.Eventf(request, corev1.EventTypeNormal, "Created", "Created Secret %q", secretCopy.Name)
	} else {
		r.Recorder.Eventf(request, corev1.EventTypeNormal, "Updated", "Updated Secret %q", secretCopy.Name)
	}
	return ctrl.Result{}, nil
}

func (r *RequestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(&corev1.Secret{}, jobOwnerKey, func(rawObj runtime.Object) []string {
		secret := rawObj.(*corev1.Secret)
		owner := metav1.GetControllerOf(secret)
		if owner == nil {
			return nil
		}
		if owner.APIVersion != apiGVStr || owner.Kind != "Request" {
			return nil
		}
		return []string{owner.Name}
	}); err != nil {
		return err
	}

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
			if err := r.List(ctx, &intents, client.InNamespace(a.Meta.GetNamespace()), client.MatchingField(".metadata.secretName", a.Meta.GetName())); err != nil {
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
