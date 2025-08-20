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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	"time"

	blogv1alpha1 "github.com/Edward0x/blog-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// BlogReconciler reconciles a Blog object
type BlogReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=blog.678645.xyz,resources=blogs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=blog.678645.xyz,resources=blogs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=blog.678645.xyz,resources=blogs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Blog object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *BlogReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = klog.FromContext(ctx)

	// TODO(user): your logic here
	var blog blogv1alpha1.Blog
	if err := r.Get(ctx, req.NamespacedName, &blog); err != nil {
		if errors.IsNotFound(err) {
			// CR 被删除了，我们不需要再做什么了，因为关联资源会被 GC 自动清理（如果设置了 OwnerReference）
			println("MyResource resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// 读取失败，可能是 API Server 暂时的网络问题，返回错误，稍后重试
		klog.Error(err, "Failed to get MyResource")
		return ctrl.Result{}, err
	}

	// --- 协调逻辑 ---

	// **阶段 1: 检查前置依赖**
	// 检查是否已手动创建 TLS secret 和 ImagePullSecret
	// 检查是否已手动安装 MySQL 和 Redis (通过检查 Service)

	// **阶段 2: 协调后端应用**
	// 协调后端 ConfigMap
	backendCm := r.buildBackendConfigMap(&blog)
	if err := r.reconcileConfigMap(ctx, backendCm); err != nil {
		klog.Error(err, "Failed to reconcile Backend ConfigMap")
		return ctrl.Result{}, err
	}

	// 协调后端 Service
	backendSvc := r.buildBackendService(&blog)
	if err := r.reconcileService(ctx, backendSvc); err != nil {
		klog.Error(err, "Failed to reconcile Backend Service")
		return ctrl.Result{}, err
	}

	// 协调后端 Deployment
	backendDep := r.buildBackendDeployment(&blog)
	if err := r.reconcileDeployment(ctx, backendDep); err != nil {
		klog.Error(err, "Failed to reconcile Backend Deployment")
		return ctrl.Result{}, err
	}

	// **阶段 3: 检查后端状态，实现依赖关系**
	currentBackendDep := &appsv1.Deployment{}
	err := r.Get(ctx, client.ObjectKey{Name: "blog-backend", Namespace: blog.Namespace}, currentBackendDep)
	if err != nil {
		klog.Error(err, "Failed to get current Backend Deployment for status check")
		return ctrl.Result{}, err
	}
	if currentBackendDep.Status.ReadyReplicas < 1 {
		klog.Info("Waiting for Backend Deployment to become ready...")
		// 更新状态... (省略)
		return ctrl.Result{RequeueAfter: 15 * time.Second}, nil
	}
	klog.Info("Backend is ready. Proceeding with frontend.")

	// **阶段 4: 协调前端应用 (只有后端 Ready 后才执行)**
	// 协调前端 ConfigMap
	frontendCm := r.buildFrontendConfigMap(&blog)
	if err := r.reconcileConfigMap(ctx, frontendCm); err != nil {
		klog.Error(err, "Failed to reconcile Frontend ConfigMap")
		return ctrl.Result{}, err
	}

	// 协调稳定版前端 (v1)
	frontendV1Dep := r.buildFrontendDeployment(&blog, blog.Spec.FrontendImage, "v1.0.1") // 硬编码版本标签
	if err := r.reconcileDeployment(ctx, frontendV1Dep); err != nil {
		klog.Error(err, "Failed to reconcile Frontend v1 Deployment")
		return ctrl.Result{}, err
	}
	frontendV1Svc := r.buildFrontendService(&blog, "v1.0.1")
	if err := r.reconcileService(ctx, frontendV1Svc); err != nil {
		klog.Error(err, "Failed to reconcile Frontend v1 Service")
		return ctrl.Result{}, err
	}

	// 协调金丝雀前端 (v2)，如果指定了的话
	if blog.Spec.FrontendCanaryImage != "" {
		frontendV2Dep := r.buildFrontendDeployment(&blog, blog.Spec.FrontendCanaryImage, "v2.0.0")
		if err := r.reconcileDeployment(ctx, frontendV2Dep); err != nil {
			klog.Error(err, "Failed to reconcile Frontend v2 Deployment")
			return ctrl.Result{}, err
		}
		frontendV2Svc := r.buildFrontendService(&blog, "v2.0.0")
		if err := r.reconcileService(ctx, frontendV2Svc); err != nil {
			klog.Error(err, "Failed to reconcile Frontend v2 Service")
			return ctrl.Result{}, err
		}
	}

	// **阶段 5: 协调网络资源**
	// 协调 Gateway
	gateway := r.buildGateway(&blog)
	if err := r.reconcileGateway(ctx, gateway); err != nil {
		klog.Error(err, "Failed to reconcile Gateway")
		return ctrl.Result{}, err
	}

	// 协调 HTTPRoute
	httpRoute := r.buildHTTPRoute(&blog)
	if err := r.reconcileHTTPRoute(ctx, httpRoute); err != nil {
		klog.Error(err, "Failed to reconcile HTTPRoute")
		return ctrl.Result{}, err
	}

	klog.Info("All resources have been reconciled successfully.")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BlogReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&blogv1alpha1.Blog{}).
		//Named("blog").
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&gatewayv1.HTTPRoute{}).
		// Gateway 在 istio-system, 不能用 Owns，需要用 Watches
		// Watches(&gatewayv1.Gateway{}, ...). // 暂时简化，不 watch Gateway
		Complete(r)
}

func (r *BlogReconciler) reconcileConfigMap(ctx context.Context, cm *corev1.ConfigMap) error {
	existing := &corev1.ConfigMap{}

	err := r.Get(ctx, client.ObjectKeyFromObject(cm), existing)
	if err != nil {
		if errors.IsNotFound(err) {
			klog.FromContext(ctx).Info("Creating a new ConfigMap", "ConfigMap.Name", cm.Name)
			return r.Create(ctx, cm)
		}
		return err
	}

	klog.V(1).Info("ConfigMap spec is already in sync", "ConfigMap.Name", existing.Name) // V(1) for verbose logging
	return nil
}

func (r *BlogReconciler) reconcileDeployment(ctx context.Context, desired *appsv1.Deployment) error {
	existing := &appsv1.Deployment{}

	err := r.Get(ctx, client.ObjectKeyFromObject(desired), existing)
	if err != nil {
		if errors.IsNotFound(err) {
			klog.FromContext(ctx).Info("Creating a new Deployment", "Deployment.Name", desired.Name)
			return r.Create(ctx, desired)
		}
		return err
	}
	// 3. 如果存在，则进行智能更新
	// 比较关心的字段。如果都不需要更新，就什么都不做，直接返回。
	needsUpdate := false

	// 比较副本数
	if existing.Spec.Replicas != nil && desired.Spec.Replicas != nil && *existing.Spec.Replicas != *desired.Spec.Replicas {
		klog.Info("Deployment replicas mismatch", "existing", *existing.Spec.Replicas, "desired", *desired.Spec.Replicas)
		existing.Spec.Replicas = desired.Spec.Replicas
		needsUpdate = true
	}

	// 比较镜像
	if len(existing.Spec.Template.Spec.Containers) > 0 && len(desired.Spec.Template.Spec.Containers) > 0 {
		if existing.Spec.Template.Spec.Containers[0].Image != desired.Spec.Template.Spec.Containers[0].Image {
			klog.Info("Deployment image mismatch", "existing", existing.Spec.Template.Spec.Containers[0].Image, "desired", desired.Spec.Template.Spec.Containers[0].Image)
			existing.Spec.Template.Spec.Containers[0].Image = desired.Spec.Template.Spec.Containers[0].Image
			needsUpdate = true
		}
	}

	// 在这里可以继续添加对其他字段的比较，比如 Ports, Env, Resources 等

	// 4. 如果有任何字段需要更新，才执行 Update
	if needsUpdate {
		klog.Info("Updating existing Deployment", "Deployment.Name", existing.Name)
		// 注意：Update 函数需要传入我们修改过的 existing 对象
		return r.Update(ctx, existing)
	}

	klog.V(1).Info("Deployment spec is already in sync", "Deployment.Name", existing.Name) // V(1) for verbose logging
	return nil
}

func (r *BlogReconciler) reconcileService(ctx context.Context, svc *corev1.Service) error {
	existing := &corev1.Service{}

	err := r.Get(ctx, client.ObjectKeyFromObject(svc), existing)
	if err != nil {
		if errors.IsNotFound(err) {
			klog.FromContext(ctx).Info("Creating a new Service", "Service.Name", svc.Name)
			return r.Create(ctx, svc)
		}
		return err
	}

	klog.V(1).Info("Service spec is already in sync", "Service.Name", existing.Name) // V(1) for verbose logging
	return nil
}

func (r *BlogReconciler) reconcileGateway(ctx context.Context, gateway *gatewayv1.Gateway) error {
	existing := &gatewayv1.Gateway{}

	err := r.Get(ctx, client.ObjectKeyFromObject(gateway), existing)
	if err != nil {
		if errors.IsNotFound(err) {
			klog.FromContext(ctx).Info("Creating a new Gateway", "Gateway.Name", gateway.Name)
			return r.Create(ctx, gateway)
		}
		return err
	}

	klog.V(1).Info("Gateway spec is already in sync", "Gateway.Name", existing.Name) // V(1) for verbose logging
	return nil
}

func (r *BlogReconciler) reconcileHTTPRoute(ctx context.Context, route *gatewayv1.HTTPRoute) error {
	existing := &gatewayv1.HTTPRoute{}

	err := r.Get(ctx, client.ObjectKeyFromObject(route), existing)
	if err != nil {
		if errors.IsNotFound(err) {
			klog.FromContext(ctx).Info("Creating a new HTTPRoute", "HTTPRoute.Name", route.Name)
			return r.Create(ctx, route)
		}
		return err
	}

	klog.V(1).Info("HTTPRoute spec is already in sync", "HTTPRoute.Name", existing.Name) // V(1) for verbose logging
	return nil
}
