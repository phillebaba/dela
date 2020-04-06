package controllers

import (
	"context"
	"errors"
	"fmt"
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
	//"sigs.k8s.io/controller-runtime/pkg/handler"
	//"sigs.k8s.io/controller-runtime/pkg/reconcile"
	//"sigs.k8s.io/controller-runtime/pkg/source"

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

	defer func() {
		if err := r.Status().Update(ctx, shareRequest); err != nil {
			log.Error(err, "Could not update ShareRequest status")
		}
	}()

	// Get ShareIntent for ShareRequest
	shareIntentNN := types.NamespacedName{Name: shareRequest.Spec.IntentReference.Name, Namespace: shareRequest.Spec.IntentReference.Namespace}
	shareIntent := &sharev1alpha1.ShareIntent{}
	if err := r.Get(ctx, shareIntentNN, shareIntent); err != nil {
		if apierrors.IsNotFound(err) {
			shareRequest.Status.State = sharev1alpha1.SRNotFound
		}

		log.Error(err, "Could not get ShareIntent referenced ShareRequest", "ShareRequest", req.NamespacedName, "ShareIntent", shareIntentNN)
		return ctrl.Result{}, err
	}

	// Check if ShareRequest from namespace is allowed
	matches, err := matchesAllowedNamespace(shareRequest.Namespace, shareIntent.Spec.AllowedNamespaces)
	if err != nil {
		return ctrl.Result{}, err
	}

	if matches == false {
		log.Info("ShareIntent does not allow ShareRequest from the current namespace", "Namespace", req.Namespace, "ShareIntent", shareIntentNN, "ShareRequest", req.NamespacedName)
		return ctrl.Result{Requeue: true, RequeueAfter: time.Minute * 1}, nil
	}

	// Make sure Secret destination does not already exist
	existSecret := &corev1.Secret{}
	err = r.Get(ctx, types.NamespacedName{Name: shareIntent.Spec.SecretReference, Namespace: req.Namespace}, existSecret)
	if client.IgnoreNotFound(err) != nil {
		return ctrl.Result{}, err
	}
	fmt.Print("Existing Secret: ")
	fmt.Println(existSecret)
	if err == nil && existSecret != nil {
		owner := metav1.GetControllerOf(existSecret)
		if owner == nil || owner.Kind != "ShareRequest" && owner.Name != shareRequest.Name {
			shareRequest.Status.State = sharev1alpha1.SRAlreadyExists
			log.Info("Destination Secret already exists", "Secret", existSecret.Name)
			return ctrl.Result{}, errors.New("Secret already exists")
		}
	}

	// Get Secret for ShareIntent
	secretNN := types.NamespacedName{Name: shareIntent.Spec.SecretReference, Namespace: shareIntentNN.Namespace}
	secret := &corev1.Secret{}
	if err := r.Get(ctx, secretNN, secret); err != nil {
		log.Error(err, "Could not get ShareIntents referenced Secret", "ShareIntent", shareIntentNN, "Secret", secretNN)
		return ctrl.Result{}, err
	}

	// Create copy of ShareIntents Secret
	secretCopy := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: secret.Name, Namespace: req.Namespace}}
	_, err = ctrl.CreateOrUpdate(ctx, r, secretCopy, func() error {
		secretCopy.Annotations = map[string]string{}
		secretCopy.Annotations["phillebaba.io/generated-from"] = shareRequest.Namespace + "/" + shareRequest.Name
		secretCopy.Data = secret.Data
		return controllerutil.SetControllerReference(shareRequest, secretCopy, r.Scheme)
	})

	if err != nil {
		return ctrl.Result{}, err
	}

	shareRequest.Status.State = sharev1alpha1.SRReady
	r.Log.Info("Created or Updated Secret", "Secret", secret.Namespace+"/"+secret.Name)
	return ctrl.Result{}, nil
}

func (r *ShareRequestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(&sharev1alpha1.ShareRequest{}, ".metadata.intentRef", func(rawObj runtime.Object) []string {
		shareRequest := rawObj.(*sharev1alpha1.ShareRequest)
		return []string{shareRequest.Spec.IntentReference.Name + "/" + shareRequest.Spec.IntentReference.Namespace}
	}); err != nil {
		return err
	}

	/*mapFn := handler.ToRequestsFunc(
		func(a handler.MapObject) []reconcile.Request {
			ctx := context.Background()

			var shareIntents sharev1alpha1.ShareIntentList
			if err := r.List(ctx, &shareIntents, client.InNamespace(a.Meta.GetNamespace()), client.MatchingField(".metadata.secretRef", a.Meta.GetName())); err != nil {
				return []reconcile.Request{}
			}

			requests := []reconcile.Request{}
			for _, shareIntent := range shareIntents.Items {
				var shareRequests sharev1alpha1.ShareRequestList
				if err := r.List(ctx, &shareRequests, client.MatchingField(".metadata.intentRef", shareIntent.Name+"/"+shareIntent.Namespace)); err != nil {
					return []reconcile.Request{}
				}

				for _, shareRequest := range shareRequests.Items {
					requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{
						Name:      shareRequest.Name,
						Namespace: shareRequest.Namespace,
					}})
				}
			}

			return requests
		},
	)*/

	return ctrl.NewControllerManagedBy(mgr).
		For(&sharev1alpha1.ShareRequest{}).
		Owns(&corev1.Secret{}).
		/*Watches(
			&source.Kind{Type: &corev1.Secret{}},
			&handler.EnqueueRequestsFromMapFunc{ToRequests: mapFn},
		).*/
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
