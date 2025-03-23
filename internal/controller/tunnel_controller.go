/*
Copyright 2025.

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

package controller

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	cloudflarev1beta1 "github.com/laininthewired/cloudflare-ingress-controller/api/v1beta1"
)

// TunnelReconciler reconciles a Tunnel object
type TunnelReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=cloudflare.laininthewired.github.io,resources=tunnels,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cloudflare.laininthewired.github.io,resources=tunnels/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cloudflare.laininthewired.github.io,resources=tunnels/finalizers,verbs=update

// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;update;patch
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Tunnel object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.0/pkg/reconcile

func (r *TunnelReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// apiToken, _, err := r.getAPITokenFromSecret(ctx)
	// if err != nil {
	// 	return ctrl.Result{}, err
	// }

	// tunnelName := "my-tunnel" // 例として固定値

	var tunnel cloudflarev1beta1.Tunnel

	if err := r.Get(ctx, req.NamespacedName, &tunnel); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if tunnel.Status.Phase == "Running" {
		logger.Info("Tunnel already running", "tunnel", tunnel.Spec.TunnelID)
		return ctrl.Result{}, nil
	}
	if tunnel.Spec.TunnelID == "" {
	}

	// TODO(user): your logic here
	// API トークンは Secret から取得する

	return ctrl.Result{}, nil
}

// func (r *TunnelReconciler) reconcileSecret(ctx context.Context, tunnel cloudflarev1beta1.Tunnel) error {

// }

// SetupWithManager sets up the controller with the Manager.
func (r *TunnelReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cloudflarev1beta1.Tunnel{}).
		Named("tunnel").
		Complete(r)
}

func (r *TunnelReconciler) getAPITokenFromSecret(ctx context.Context) (string, error) {
	// ここでは固定値として設定しています。必要に応じて CRD の Spec や ConfigMap 等から動的に取得してください。
	secret := &corev1.Secret{}
	secretName := "cloudflare-api-token"
	secretNamespace := "default"
	if err := r.Get(ctx, client.ObjectKey{Namespace: secretNamespace, Name: secretName}, secret); err != nil {
		return "", fmt.Errorf("failed to get secret %s/%s: %w", secretNamespace, secretName, err)
	}
	apiTokenBytes, ok := secret.Data["apiToken"]
	if !ok {
		return "", fmt.Errorf("secret %s/%s does not contain key 'apiToken'", secretNamespace, secretName)
	}
	apiToken := strings.TrimSpace(string(apiTokenBytes))
	return apiToken, nil
}
