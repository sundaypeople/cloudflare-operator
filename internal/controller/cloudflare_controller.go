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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	appsv1apply "k8s.io/client-go/applyconfigurations/apps/v1"
	corev1apply "k8s.io/client-go/applyconfigurations/core/v1"
	metav1apply "k8s.io/client-go/applyconfigurations/meta/v1"
	"k8s.io/utils/pointer"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	cloudflarev1beta1 "github.com/laininthewired/cloudflare-ingress-controller/api/v1beta1"
	"gopkg.in/yaml.v3"
)

// CloudflareReconciler reconciles a Cloudflare object
type CloudflareReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// IngressRule は単一のIngressルールを表します。
type IngressRule struct {
	// Hostname はホスト名です。必須フィールドです。
	Hostname string `json:"hostname,omitempty" yaml:"hostname,omitempty"`

	// Service はサービスのURLです。必須フィールドです。
	Service string `json:"service" yaml:"service"`
}

// CloudflareSpec はクラウドフレアの仕様を表します。
type CloudflareConfig struct {
	// Ingress はIngressルールのリストです。

	Tunnel string `json:"tunnel" yaml:"tunnel"`

	Metrics string `json:"metrics" yaml:"metrics"`

	Ingress []IngressRule `json:"ingress,omitempty" yaml:"ingress,omitempty"`

	// Replicas はレプリカの数です。
	// Replicas int32 `json:"replicas,omitempty" yaml:"replicas,omitempty"`
	// TunnelID はトンネルのIDです。必須フィールドです。

	CredentialsFile string `json:"credentials-file" yaml:"credentials-file"`
}

// +kubebuilder:rbac:groups=cloudflare.laininthewired.github.io,resources=cloudflares,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cloudflare.laininthewired.github.io,resources=cloudflares/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cloudflare.laininthewired.github.io,resources=cloudflares/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;update;patch
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Cloudflare object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.0/pkg/reconcile

func (r *CloudflareReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var cf cloudflarev1beta1.Cloudflare
	err := r.Get(ctx, req.NamespacedName, &cf)
	if errors.IsNotFound(err) {
		return ctrl.Result{}, nil
	}
	if err != nil {
		logger.Error(err, "unable to get Cloudflare", "name", req.NamespacedName)
		return ctrl.Result{}, err
	}
	if !cf.ObjectMeta.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	err = r.reconcileConfigMap(ctx, cf)
	if err != nil {
		result, err2 := r.updateStatus(ctx, cf)
		logger.Error(err2, "unable to update status")
		return result, err
	}
	err = r.reconcileDeployment(ctx, cf)
	if err != nil {
		result, err2 := r.updateStatus(ctx, cf)
		logger.Error(err2, "unable to update status")
		return result, err
	}

	// TODO(user): your logic here

	return ctrl.Result{}, nil
}
func (r *CloudflareReconciler) reconcileConfigMap(ctx context.Context, cloudflare cloudflarev1beta1.Cloudflare) error {
	logger := log.FromContext(ctx)

	cm := &corev1.ConfigMap{}
	cm.SetNamespace(cloudflare.Namespace)
	cm.SetName("cloudflare-" + cloudflare.Name)

	var ingressRules []IngressRule

	for _, content := range cloudflare.Spec.Ingress {
		ingressRule := IngressRule{
			Hostname: content.Hostname, // Hostnameが空の場合は省略可能
			Service:  content.Service,
		}
		ingressRules = append(ingressRules, ingressRule)
	}
	ingressRule := IngressRule{
		// Hostname: "" // Hostnameが空の場合は省略可能
		Service: "http_status:404",
	}
	ingressRules = append(ingressRules, ingressRule)
	// 構造体をYAMLにシリアライズ
	spec := CloudflareConfig{
		Tunnel:          cloudflare.Spec.TunnelID, // 必須フィールドを設定
		CredentialsFile: "/etc/cloudflared/creds/credentials.json",
		Ingress:         ingressRules,
		Metrics:         "0.0.0.0:2000",
	}

	yamlBytes, err := yaml.Marshal(&spec)
	if err != nil {
		logger.Error(err, "configmap marshal error")
	}
	// YAML文字列を出力
	yamlString := string(yamlBytes)
	fmt.Println(yamlString)

	op, err := ctrl.CreateOrUpdate(ctx, r.Client, cm, func() error {
		if cm.Data == nil {
			cm.Data = make(map[string]string)
		}

		cm.Data["config.yaml"] = yamlString

		return ctrl.SetControllerReference(&cloudflare, cm, r.Scheme)
	})

	if err != nil {
		logger.Error(err, "unable to create or update ConfigMap")
		return err
	}

	if op != controllerutil.OperationResultNone {
		logger.Info("reconcile ConfigMap successfully", "op", op)
	}

	return nil
}

// func (r *CloudflareReconciler) reconcileSecret(ctx context.Context, cloudflare cloudflarev1beta1.Cloudflare) error {
// 	logger := log.FromContext(ctx)
// 	sc := &corev1.Secret{}
// 	sc.SetName("cloudflare-" + cloudflare.Name)
// 	sc.SetNamespace(cloudflare.Namespace)
// 	logger.Info("test")

// 	return nil
// }

func (r *CloudflareReconciler) reconcileDeployment(ctx context.Context, cloudlfare cloudflarev1beta1.Cloudflare) error {
	logger := log.FromContext(ctx)
	depName := "cloudflare-" + cloudlfare.Name
	cloudflareimage := "cloudflare/cloudflared:2025.1.0"
	// cloudflareimage := "nginx:1.14.2"

	// dep := appsv1apply.Deployment(depName, mdView.Namespace).
	// 			WithLabels(map[string]string{
	// 				"app.kubernetes.io/name":       "cloudflare",
	// 				"app.kubernetes.io/instance":   Cloudflare.Name,
	// 				"app.kubernetes.io/created-by": "cloudflared-operator-controller-manager",
	// 			}).
	// 			WithSpec(appsv1apply.DeploymentSpec().
	// 				WithReplicas(mdView.Spec.Replicas).
	// 				WithSelector(metav1apply.LabelSelector().WithMatchLabels(map[string]string{
	// 					"app.kubernetes.io/name":       "cloudflare",
	// 					"app.kubernetes.io/instance":   Cloudflare.Name,
	// 					"app.kubernetes.io/created-by": "cloudflared-operator-controller-manager",
	// 				})).
	// 				WithTemplate(corev1apply.PodTemplateSpec().
	// 					WithLabels(map[string]string{
	// 						"app.kubernetes.io/name":       "cloudflare",
	// 						"app.kubernetes.io/instance":   Cloudflare.Name,
	// 						"app.kubernetes.io/created-by": "cloudflared-operator-controller-manager",
	// 					}).
	// 					WithSpec(corev1apply.PodSpec().
	// 					WithContainers(corev1apply.Container().
	// 						WithName("cloudflare").
	// 						WithImage(viewerImage).
	// 						WithImagePullPolicy(corev1.PullIfNotPresent).
	// 						WithCommand("cloudflare").
	// 						WithArgs("serve", "--hostname", "0.0.0.0").
	// 						WithVolumeMounts(corev1apply.VolumeMount().
	// 							WithName("markdowns").
	// 							WithMountPath("/book/src"),
	// 						).
	// 						WithPorts(corev1apply.ContainerPort().
	// 							WithName("http").
	// 							WithProtocol(corev1.ProtocolTCP).
	// 							WithContainerPort(3000),
	// 						).
	// 						WithLivenessProbe(corev1apply.Probe().
	// 							WithHTTPGet(corev1apply.HTTPGetAction().
	// 								WithPort(intstr.FromString("http")).
	// 								WithPath("/").
	// 								WithScheme(corev1.URISchemeHTTP),
	// 							),
	// 						).
	// 						WithReadinessProbe(corev1apply.Probe().
	// 							WithHTTPGet(corev1apply.HTTPGetAction().
	// 								WithPort(intstr.FromString("http")).
	// 								WithPath("/").
	// 								WithScheme(corev1.URISchemeHTTP),
	// 							),
	// 						),
	// 					).
	owner, err := controllerReference(cloudlfare, r.Scheme)
	if err != nil {
		return err
	}
	deployment := appsv1apply.Deployment(depName, cloudlfare.Namespace).
		WithLabels(map[string]string{
			"app.kubernetes.io/name":       "cloudflare",
			"app.kubernetes.io/instance":   cloudlfare.Name,
			"app.kubernetes.io/created-by": "cloudflared-operator-controller-manager",
		}).
		WithOwnerReferences(owner).
		WithSpec(appsv1apply.DeploymentSpec().
			WithReplicas(cloudlfare.Spec.Replicas).
			WithSelector(metav1apply.LabelSelector().
				WithMatchLabels(map[string]string{
					"app.kubernetes.io/name":       "cloudflare",
					"app.kubernetes.io/instance":   cloudlfare.Name,
					"app.kubernetes.io/created-by": "cloudflared-operator-controller-manager",
				}),
			).
			WithTemplate(corev1apply.PodTemplateSpec().
				WithLabels(map[string]string{
					"app.kubernetes.io/name":       "cloudflare",
					"app.kubernetes.io/instance":   cloudlfare.Name,
					"app.kubernetes.io/created-by": "cloudflared-operator-controller-manager",
				}).
				WithSpec(corev1apply.PodSpec().
					WithContainers(
						corev1apply.Container().
							WithName("cloudflared").
							WithImage(cloudflareimage).
							WithArgs(
								"tunnel",
								"--config",
								"/etc/cloudflared/config/config.yaml",
								"--http2-origin",
								"run",
							).
							// WithCommand("/bin/sh").
							// WithArgs("-c", "sleep 3600").
							WithLivenessProbe(
								corev1apply.Probe().
									WithHTTPGet(
										corev1apply.HTTPGetAction().
											WithPath("/ready").
											WithPort(intstr.FromInt(2000)),
									).
									WithFailureThreshold(1).
									WithInitialDelaySeconds(10).
									WithPeriodSeconds(10),
							).
							// WithTTY(true).   // TTY を有効化
							// WithStdin(true). // Stdin を有効化
							WithVolumeMounts(
								corev1apply.VolumeMount().
									WithName("config").
									WithMountPath("/etc/cloudflared/config").
									WithReadOnly(true),
								corev1apply.VolumeMount().
									WithName("creds").
									WithMountPath("/etc/cloudflared/creds").
									WithReadOnly(true),
							),
					).
					WithVolumes(
						corev1apply.Volume().
							WithName("creds").
							WithSecret(
								corev1apply.SecretVolumeSource().
									WithSecretName("tunnel-credentials"),
							),
						corev1apply.Volume().
							WithName("config").
							WithConfigMap(
								corev1apply.ConfigMapVolumeSource().
									WithName("cloudflare-"+cloudlfare.Name).
									WithItems(
										corev1apply.KeyToPath().
											WithKey("config.yaml").
											WithPath("config.yaml"),
									),
							),
					),
				),
			),
		)
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(deployment)
	if err != nil {
		return err
	}
	patch := &unstructured.Unstructured{
		Object: obj,
	}

	var current appsv1.Deployment
	err = r.Get(ctx, client.ObjectKey{Namespace: cloudlfare.Namespace, Name: depName}, &current)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	currApplyConfig, err := appsv1apply.ExtractDeployment(&current, "cloudflared-operator-controller-manager")
	if err != nil {
		return err
	}

	if equality.Semantic.DeepEqual(deployment, currApplyConfig) {
		return nil
	}

	err = r.Patch(ctx, patch, client.Apply, &client.PatchOptions{
		FieldManager: "cloudflared-operator-controller-manager",
		Force:        pointer.Bool(true),
	})

	if err != nil {
		logger.Error(err, "unable to create or update Deployment")
		return err
	}
	logger.Info("reconcile Deployment successfully", "name", cloudlfare.Name)
	return nil
}
func (r *CloudflareReconciler) updateStatus(ctx context.Context, Cloudflare cloudflarev1beta1.Cloudflare) (ctrl.Result, error) {
	meta.SetStatusCondition(&Cloudflare.Status.Conditions, metav1.Condition{
		Type:   cloudflarev1beta1.TypeCloudflareViewAvailable,
		Status: metav1.ConditionTrue,
		Reason: "OK",
	})
	meta.SetStatusCondition(&Cloudflare.Status.Conditions, metav1.Condition{
		Type:   cloudflarev1beta1.TypeCloudflareViewDegraded,
		Status: metav1.ConditionFalse,
		Reason: "OK",
	})

	var cm corev1.ConfigMap
	err := r.Get(ctx, client.ObjectKey{Namespace: Cloudflare.Namespace, Name: "cloudflare-" + Cloudflare.Name}, &cm)
	if errors.IsNotFound(err) {
		meta.SetStatusCondition(&Cloudflare.Status.Conditions, metav1.Condition{
			Type:    cloudflarev1beta1.TypeCloudflareViewDegraded,
			Status:  metav1.ConditionTrue,
			Reason:  "Reconciling",
			Message: "ConfigMap not found",
		})
		meta.SetStatusCondition(&Cloudflare.Status.Conditions, metav1.Condition{
			Type:   cloudflarev1beta1.TypeCloudflareViewAvailable,
			Status: metav1.ConditionFalse,
			Reason: "Reconciling",
		})
	} else if err != nil {
		return ctrl.Result{}, err
	}

	// var svc corev1.Service
	// err = r.Get(ctx, client.ObjectKey{Namespace: Cloudflare.Namespace, Name: "cloudflare-" + Cloudflare.Name}, &svc)
	// if errors.IsNotFound(err) {
	// 	meta.SetStatusCondition(&Cloudflare.Status.Conditions, metav1.Condition{
	// 		Type:    viewv1.TypeMarkdownViewDegraded,
	// 		Status:  metav1.ConditionTrue,
	// 		Reason:  "Reconciling",
	// 		Message: "Service not found",
	// 	})
	// 	meta.SetStatusCondition(&Cloudflare.Status.Conditions, metav1.Condition{
	// 		Type:   viewv1.TypeMarkdownViewAvailable,
	// 		Status: metav1.ConditionFalse,
	// 		Reason: "Reconciling",
	// 	})
	// } else if err != nil {
	// 	return ctrl.Result{}, err
	// }

	var dep appsv1.Deployment
	err = r.Get(ctx, client.ObjectKey{Namespace: Cloudflare.Namespace, Name: "viewer-" + Cloudflare.Name}, &dep)
	if errors.IsNotFound(err) {
		meta.SetStatusCondition(&Cloudflare.Status.Conditions, metav1.Condition{
			Type:    cloudflarev1beta1.TypeCloudflareViewDegraded,
			Status:  metav1.ConditionTrue,
			Reason:  "Reconciling",
			Message: "Deployment not found",
		})
		meta.SetStatusCondition(&Cloudflare.Status.Conditions, metav1.Condition{
			Type:   cloudflarev1beta1.TypeCloudflareViewAvailable,
			Status: metav1.ConditionFalse,
			Reason: "Reconciling",
		})
	} else if err != nil {
		return ctrl.Result{}, err
	}

	result := ctrl.Result{}
	if dep.Status.AvailableReplicas == 0 {
		meta.SetStatusCondition(&Cloudflare.Status.Conditions, metav1.Condition{
			Type:    cloudflarev1beta1.TypeCloudflareViewAvailable,
			Status:  metav1.ConditionFalse,
			Reason:  "Unavailable",
			Message: "AvailableReplicas is 0",
		})
		result = ctrl.Result{Requeue: true}
	}

	err = r.Status().Update(ctx, &Cloudflare)
	return result, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *CloudflareReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cloudflarev1beta1.Cloudflare{}).
		Named("cloudflare").
		Complete(r)
}
func controllerReference(cloudflare cloudflarev1beta1.Cloudflare, scheme *runtime.Scheme) (*metav1apply.OwnerReferenceApplyConfiguration, error) {
	gvk, err := apiutil.GVKForObject(&cloudflare, scheme)
	if err != nil {
		return nil, err
	}
	ref := metav1apply.OwnerReference().
		WithAPIVersion(gvk.GroupVersion().String()).
		WithKind(gvk.Kind).
		WithName(cloudflare.Name).
		WithUID(cloudflare.GetUID()).
		WithBlockOwnerDeletion(true).
		WithController(true)
	return ref, nil
}

// make docker-build
// kind load docker-image controller:latest
// kubectl logs -n cloudflared-operator-system deployments/cloudflared-operator-controller-manager -c manager -f
// kubectl rollout restart -n cloudflared-operator-system deployment cloudflared-operator-controller-manager
// kc delete cloudflares cloudflare-sample
