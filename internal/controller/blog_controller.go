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

	// 为后续日志添加上下文
	//log = log.WithValues("blog", req.NamespacedName)

	// --- 协调逻辑 ---

	// **阶段 1: 检查前置依赖**
	// 检查用户是否已手动创建 TLS secret 和 ImagePullSecret
	// 检查用户是否已手动安装 MySQL 和 Redis (通过检查 Service)
	// (这部分逻辑暂时省略以保持核心流程清晰，但生产级 Operator 需要)

	// **阶段 2: 协调后端应用**
	// 协调后端 ConfigMap
	backendCm := r.buildBackendConfigMap(&blog)
	if err := r.CreateOrUpdate(ctx, backendCm); err != nil {
		klog.Error(err, "Failed to reconcile Backend ConfigMap")
		return ctrl.Result{}, err
	}

	// 协调后端 Service
	backendSvc := r.buildBackendService(&blog)
	if err := r.CreateOrUpdate(ctx, backendSvc); err != nil {
		klog.Error(err, "Failed to reconcile Backend Service")
		return ctrl.Result{}, err
	}

	// 协调后端 Deployment
	backendDep := r.buildBackendDeployment(&blog)
	if err := r.CreateOrUpdateDeployment(ctx, backendDep); err != nil {
		klog.Error(err, "Failed to reconcile Backend Deployment")
		return ctrl.Result{}, err
	}

	// **阶段 3: 检查后端状态，实现依赖关系**
	if backendDep.Status.ReadyReplicas < 1 {
		klog.Info("Waiting for Backend Deployment to become ready...")
		// 更新状态... (省略)
		return ctrl.Result{RequeueAfter: 15 * time.Second}, nil
	}
	klog.Info("Backend is ready. Proceeding with frontend.")

	// **阶段 4: 协调前端应用 (只有后端 Ready 后才执行)**
	// 协调前端 ConfigMap
	frontendCm := r.buildFrontendConfigMap(&blog)
	if err := r.CreateOrUpdate(ctx, frontendCm); err != nil {
		klog.Error(err, "Failed to reconcile Frontend ConfigMap")
		return ctrl.Result{}, err
	}

	// 协调稳定版前端 (v1)
	frontendV1Dep := r.buildFrontendDeployment(&blog, blog.Spec.FrontendImage, "v1.0.1") // 硬编码版本标签
	if err := r.CreateOrUpdateDeployment(ctx, frontendV1Dep); err != nil {
		klog.Error(err, "Failed to reconcile Frontend v1 Deployment")
		return ctrl.Result{}, err
	}
	frontendV1Svc := r.buildFrontendService(&blog, "v1.0.1")
	if err := r.CreateOrUpdate(ctx, frontendV1Svc); err != nil {
		klog.Error(err, "Failed to reconcile Frontend v1 Service")
		return ctrl.Result{}, err
	}

	// 协调金丝雀前端 (v2)，如果指定了的话
	if blog.Spec.FrontendCanaryImage != "" {
		frontendV2Dep := r.buildFrontendDeployment(&blog, blog.Spec.FrontendCanaryImage, "v2.0.0")
		if err := r.CreateOrUpdateDeployment(ctx, frontendV2Dep); err != nil {
			klog.Error(err, "Failed to reconcile Frontend v2 Deployment")
			return ctrl.Result{}, err
		}
		frontendV2Svc := r.buildFrontendService(&blog, "v2.0.0")
		if err := r.CreateOrUpdate(ctx, frontendV2Svc); err != nil {
			klog.Error(err, "Failed to reconcile Frontend v2 Service")
			return ctrl.Result{}, err
		}
	}

	// **阶段 5: 协调网络资源**
	// 协调 Gateway
	gateway := r.buildGateway(&blog)
	if err := r.CreateOrUpdate(ctx, gateway); err != nil {
		klog.Error(err, "Failed to reconcile Gateway")
		return ctrl.Result{}, err
	}

	// 协调 HTTPRoute
	httpRoute := r.buildHTTPRoute(&blog)
	if err := r.CreateOrUpdate(ctx, httpRoute); err != nil {
		klog.Error(err, "Failed to reconcile HTTPRoute")
		return ctrl.Result{}, err
	}

	klog.Info("All resources have been reconciled successfully.")

	return ctrl.Result{}, nil
}

// **自定义的 CreateOrUpdate 辅助函数**
// Kubebuilder 默认没有 CreateOrUpdate，我们需要自己写一个简单的包装
func (r *BlogReconciler) CreateOrUpdate(ctx context.Context, obj client.Object) error {
	err := r.Get(ctx, client.ObjectKeyFromObject(obj), obj)
	if err != nil {
		if errors.IsNotFound(err) {
			return r.Create(ctx, obj)
		}
		return err
	}
	return r.Update(ctx, obj)
}

// CreateOrUpdateDeployment 专门处理 Deployment，因为其 spec 经常变化
func (r *BlogReconciler) CreateOrUpdateDeployment(ctx context.Context, desired *appsv1.Deployment) error {
	existing := &appsv1.Deployment{}

	err := r.Get(ctx, client.ObjectKeyFromObject(desired), existing)
	if err != nil {
		if errors.IsNotFound(err) {
			klog.FromContext(ctx).Info("Creating a new Deployment", "Deployment.Name", desired.Name)
			return r.Create(ctx, desired)
		}
		return err
	}
	//if !reflect.DeepEqual(desired.Spec, existing.Spec) {
	//	klog.FromContext(ctx).Info("Updating existing Deployment", "Deployment.Name", desired.Name)
	//	existing.Spec = desired.Spec
	//	return r.Update(ctx, existing)
	//}
	//return nil

	// 简单的更新策略：只更新我们关心的字段，避免与 K8s 其他控制器冲突
	existing.Spec.Replicas = desired.Spec.Replicas
	existing.Spec.Template.Spec.Containers[0].Image = desired.Spec.Template.Spec.Containers[0].Image
	return r.Update(ctx, existing)
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
