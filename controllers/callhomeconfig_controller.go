/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	arlonv1 "github.com/arlonproj/arlon/api/v1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"time"
)

// CallHomeConfigReconciler reconciles a CallHomeConfig object
type CallHomeConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const (
	retrySeconds = 10
)

//+kubebuilder:rbac:groups=core.arlon.io,resources=callhomeconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core.arlon.io,resources=callhomeconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core.arlon.io,resources=callhomeconfigs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the CallHomeConfig object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *CallHomeConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("callhomeconfig", req.NamespacedName)
	log.V(1).Info("arlon callhomeconfig")
	var chc arlonv1.CallHomeConfig

	if err := r.Get(ctx, req.NamespacedName, &chc); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("callhomeconfig is gone -- ok")
			return ctrl.Result{}, nil
		}
		log.Info(fmt.Sprintf("unable to get callhomeconfig (%s) ... requeuing", err))
		return ctrl.Result{Requeue: true}, nil
	}
	if chc.Status.State == "complete" {
		log.V(1).Info("callhomeconfig is already complete")
		return ctrl.Result{}, nil
	}
	if chc.Status.State == "error" {
		log.V(1).Info("callhomeconfig is already in error state")
		return ctrl.Result{}, nil
	}
	var secret corev1.Secret
	secretNamespacedName := types.NamespacedName{
		Namespace: req.NamespacedName.Namespace,
		Name:      chc.Spec.KubeconfigSecretName,
	}
	if err := r.Get(ctx, secretNamespacedName, &secret); err != nil {
		if apierrors.IsNotFound(err) {
			return retryLater(r, log, &chc, "kubeconfig secret",
				chc.Spec.KubeconfigSecretName, "does not exist yet")
		}
		msg := fmt.Sprintf("failed to read secret: %s", err)
		return updateCallHomeConfigState(r, log, &chc, "error", msg, ctrl.Result{})
	}
	data := secret.Data[chc.Spec.KubeconfigSecretKeyName]
	if data == nil {
		return updateCallHomeConfigState(r, log, &chc, "error",
			fmt.Sprintf("secret subkey %s does not exist",
				chc.Spec.KubeconfigSecretKeyName), ctrl.Result{})
	}
	conf, err := clientcmd.RESTConfigFromKubeConfig(data)
	if err != nil {
		return updateCallHomeConfigState(r, log, &chc, "error",
			fmt.Sprintf("failed to read kubeconfig from secret: %s", err),
			ctrl.Result{})
	}
	clientset, err := kubernetes.NewForConfig(conf)
	if err != nil {
		return updateCallHomeConfigState(r, log, &chc, "error",
			fmt.Sprintf("failed to get clientset from config: %s", err),
			ctrl.Result{})
	}
	secretsApi := clientset.CoreV1().Secrets(chc.Spec.TargetNamespace)
	_, err = secretsApi.Get(context.Background(), chc.Spec.TargetSecretName,
		metav1.GetOptions{})
	if err == nil {
		return updateCallHomeConfigState(r, log, &chc, "complete",
			"target secret already exists",
			ctrl.Result{})
	}
	if !apierr.IsNotFound(err) {
		return retryLater(r, log, &chc, "target secret",
			chc.Spec.TargetSecretName,
			"could not be queried, workload cluster probably still unavailable")
	}
	// read the service account token
	var sa corev1.ServiceAccount
	namespacedName := types.NamespacedName{
		Namespace: req.Namespace,
		Name:      chc.Spec.ServiceAccountName,
	}
	if err := r.Get(ctx, namespacedName, &sa); err != nil {
		if apierrors.IsNotFound(err) {
			return retryLater(r, log, &chc, "serviceaccount",
				namespacedName.Name, "does not exist yet")
		}
		return updateCallHomeConfigState(r, log, &chc, "error",
			fmt.Sprintf("unexpected error getting service account: %s", err),
			ctrl.Result{})
	}
	if len(sa.Secrets) < 1 {
		return retryLater(r, log, &chc, "serviceaccount",
			namespacedName.Name, "does not have a token")
	}
	tokenSecretName := sa.Secrets[0]
	var token corev1.Secret
	if err := r.Get(ctx, types.NamespacedName{
		Namespace: req.Namespace,
		Name:      tokenSecretName.Name,
	}, &token); err != nil {
		if apierrors.IsNotFound(err) {
			return retryLater(r, log, &chc, "token secret",
				tokenSecretName.Name, "does not exist yet")
		}
		return updateCallHomeConfigState(r, log, &chc, "error",
			fmt.Sprintf("unexpected error getting service account: %s", err),
			ctrl.Result{})
	}
	cfg := clientcmdapi.NewConfig()
	clst := clientcmdapi.NewCluster()
	clst.Server = chc.Spec.ManagementClusterUrl
	clst.CertificateAuthorityData = token.Data["ca.crt"]
	if clst.CertificateAuthorityData == nil {
		return updateCallHomeConfigState(r, log, &chc, "error",
			"token secret does not have ca.crt",
			ctrl.Result{})
	}
	cfg.Clusters["management"] = clst
	user := clientcmdapi.NewAuthInfo()
	if token.Data["token"] == nil {
		return updateCallHomeConfigState(r, log, &chc, "error",
			"token secret does not have token",
			ctrl.Result{})
	}
	user.Token = string(token.Data["token"])
	cfg.AuthInfos["sa"] = user
	contx := clientcmdapi.NewContext()
	contx.Cluster = "management"
	contx.AuthInfo = "sa"
	cfg.Contexts["management"] = contx
	cfg.CurrentContext = "management"

	// Create target secret
	kubeconfigData, err := clientcmd.Write(*cfg)
	if err != nil {
		return updateCallHomeConfigState(r, log, &chc, "error",
			fmt.Sprintf("failed to serialize kubeconfig: %s", err),
			ctrl.Result{})
	}
	newSecr := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: chc.Spec.TargetSecretName,
		},
		Data: map[string][]byte{
			chc.Spec.TargetSecretKeyName: kubeconfigData,
		},
	}
	_, err = secretsApi.Create(context.Background(), &newSecr, metav1.CreateOptions{})
	if err != nil {
		return retryLater(r, log, &chc, "target secret",
			chc.Spec.TargetSecretName, fmt.Sprintf("could not be created: %s", err))
	}
	return updateCallHomeConfigState(r, log, &chc, "complete",
		"successfully created target secret", ctrl.Result{})
}

// SetupWithManager sets up the controller with the Manager.
func (r *CallHomeConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&arlonv1.CallHomeConfig{}).
		Complete(r)
}

func retryLater(
	r *CallHomeConfigReconciler,
	log logr.Logger,
	chc *arlonv1.CallHomeConfig,
	resourceType string,
	resourceName string,
	description string,
) (ctrl.Result, error) {
	msg := fmt.Sprintf("%s %s %s, retrying in %d seconds",
		resourceType, resourceName, description, retrySeconds)
	return updateCallHomeConfigState(r, log, chc, "retrying", msg,
		ctrl.Result{RequeueAfter: retrySeconds * time.Second})
}

func updateCallHomeConfigState(
	r *CallHomeConfigReconciler,
	log logr.Logger,
	chc *arlonv1.CallHomeConfig,
	state string,
	msg string,
	result ctrl.Result,
) (ctrl.Result, error) {
	if chc.Status.State == state && chc.Status.Message == msg {
		log.Info(fmt.Sprintf("%s ... already in '%s' state", msg, state))
		return result, nil
	}
	chc.Status.State = state
	chc.Status.Message = msg
	log.Info(fmt.Sprintf("%s ... setting state to '%s'", msg, chc.Status.State))
	if err := r.Status().Update(context.Background(), chc); err != nil {
		log.Error(err, "unable to update callhomeconfig status")
		return ctrl.Result{}, err
	}
	return result, nil
}
