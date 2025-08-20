package controller

import (
	// ... add all necessary imports here ...
	blogv1alpha1 "github.com/Edward0x/blog-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// labelsForBlog generates the labels for a Blog app's resources.
//func labelsForBlog(name string) map[string]string {
//	return map[string]string{
//		"app.kubernetes.io/name":       "BlogApp",
//		"app.kubernetes.io/instance":   name,
//		"app.kubernetes.io/managed-by": "BlogOperator",
//	}
//}

// --- Backend Resources ---

func (r *BlogReconciler) buildBackendConfigMap(blog *blogv1alpha1.Blog) *corev1.ConfigMap {
	var configYaml string = `
    captcha:
        height: 80
        width: 240
        length: 6
        max_skew: 0.7
        dot_count: 80
    email:
        host: smtp.163.com
        port: 465
        from: 15853696101@163.com
        nickname: Admin
        secret: XQvPQ32rneDYpLDn
        is_ssl: true
    es:
        url: https://es-cluster-es-http:9200
        username: "elastic"
        password: "4Ma3DVqhr9w9oc4WA611Ww17"
        is_console_print: true
    gaode:
        enable: false
        key: xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
    jwt:
        access_token_secret: pNCWCJQ_-13zBCXpLKCN4ljb5zl6SJUZUdVwjBdtJww=
        refresh_token_secret: 3SvXWD1dvrRvUuzHRmlyt8VtTH5tRxM67b144F09uZ4=
        access_token_expiry_time: 2h
        refresh_token_expiry_time: 7d
        issuer: go_blog
    mysql:
        host: mysql-db-primary
        port: 3306
        config: charset=utf8mb4&parseTime=True&loc=Local
        db_name: blog_db
        username: root
        password: 123456abc
        max_idle_conns: 10
        max_open_conns: 100
        log_mode: info
    cloudflare:
        bucketName: "blogtest"
        accountId: "5fc9dc2aa7b67503c60fb76703996fa7"
        accessKeyId: "1befaad0395755d5c763a28e681fca43"
        accessKeySecret: "64ad9bac2f4dec158323a823156da5b150c553d917376832d845cda0e47b03f3"
    qiniu:
        zone: z0
        bucket: xxx
        img_path: https://xxx.xxx/
        access_key: xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
        secret_key: xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
        use_https: true
        use_cdn_domains: true
    qq:
        enable: false
        app_id: "000000000"
        app_key: xxxxxxxxxxxxxxxx
        redirect_uri: http://xxx.xxx/xxx
    redis:
        enable: false
        address:
        - "redis-master:6379"
        password: ""
        db: 0
        pool_size: 10
    system:
        host: 0.0.0.0
        port: 8080
        env: release
        router_prefix: api
        use_multipoint: true
        sessions_secret: 00LVFqDHW9-Zr3F3lhSqgCY9wZ5tY2r_Y_yN2F-i4wY=
    #    oss_type: local
        oss_type: CloudFlare
    upload:
        size: 20
        path: uploads
    website:
        logo: ""
        full_logo: ""
        title: NoneBlog
        slogan: 无贡献者
        slogan_en: NoneContributors
        description: 个人测试
        version: 1.0.0
        created_at: "2025-6-15"
        icp_filing: icp备案号
        public_security_filing: 公安备案号
        bilibili_url: https://space.bilibili.com/xxx
        gitee_url: https://gitee.com/xxx
        github_url: https://github.com/EdWard0x
        name: avein
        job: 无业游民
        address: 保密
        email: avein521@gmail.com
        qq_image: ""
        wechat_image: ""
    zap:
        level: info
        filename: log/go_blog.log
        max_size: 200
        max_backups: 30
        max_age: 5
        is_console_print: true

`
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "blog-config",
			Namespace: blog.Namespace,
		},
		Data: map[string]string{
			"database.name": "blog_db",
			"mysql.host":    "mysql-db-primary",
			"mysql.port":    "3306",
			"redis.host":    "redis-master",
			"redis.port":    "6379",
			"es.host":       "es-cluster-es-http",
			"es.port":       "9200",
			"es.index":      "article_index",
			"config.yaml":   configYaml,
		},
	}
	ctrl.SetControllerReference(blog, cm, r.Scheme)
	return cm
}

func (r *BlogReconciler) buildBackendService(blog *blogv1alpha1.Blog) *corev1.Service {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "blog-backend",
			Namespace: blog.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "blog-backend",
			},
			Ports: []corev1.ServicePort{
				{
					Protocol:   corev1.ProtocolTCP,
					Port:       8080,
					TargetPort: intstr.FromInt(8080),
				},
			},
		},
	}
	ctrl.SetControllerReference(blog, svc, r.Scheme)
	return svc
}

func (r *BlogReconciler) buildBackendDeployment(blog *blogv1alpha1.Blog) *appsv1.Deployment {
	//labels := labelsForBlog(blog.Name)

	//cpu100m := resource.MustParse("100m")
	//mem128Mi := resource.MustParse("128Mi")

	com := []string{
		"sh",
		"-c",
		`# 初始化 MySQL（如果需要）
              if [ -f /data/mysql_tables_initialized ]; then
                echo "MySQL already initialized."
              else
                echo "Initializing MySQL..."
                ./app -sql
              fi
              ./app -es
              ./app -admin
              # 启动主服务
              ./app
		`,
	}
	replicas := int32(1)
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "blog-backend",
			Namespace: blog.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "blog-backend"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "blog-backend"},
				},
				Spec: corev1.PodSpec{
					SecurityContext: &corev1.PodSecurityContext{FSGroup: pointer.Int64(1000)},
					InitContainers: []corev1.Container{
						{
							Name: "check-mysql",
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceCPU: resource.Quantity{
										Format: "500m",
									},
									corev1.ResourceMemory: resource.Quantity{
										Format: "256Mi",
									},
								},
								Requests: corev1.ResourceList{
									corev1.ResourceCPU: resource.Quantity{
										Format: "100M",
									},
									corev1.ResourceMemory: resource.Quantity{
										Format: "128Mi",
									},
								},
							},
							Image: "mysql:8.0",
							Command: []string{
								"sh",
								"-c",
								`# 1. 等待 MySQL 服务就绪
              echo "Waiting for MySQL server at ${MYSQL_HOST}:${MYSQL_PORT}..."
              until mysql -h "$MYSQL_HOST" -P "$MYSQL_PORT" -u root -p"$MYSQL_ROOT_PASSWORD" -e "SELECT 1" >/dev/null 2>&1; do
                echo "Waiting for MySQL...";
                sleep 2;
              done
              echo "MySQL server is ready."

              # 3. 检查特定表是否存在
              TABLE_EXISTS=$(mysql -h "$MYSQL_HOST" -P "$MYSQL_PORT" -u root -p"$MYSQL_ROOT_PASSWORD" -N -s -r -e \
                "SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA = '${TARGET_DB}' AND TABLE_NAME = '${TARGET_TABLE}';")
              
              if [ "$TABLE_EXISTS" -gt 0 ]; then
                echo "Table '${TARGET_TABLE}' in database '${TARGET_DB}' already exists. Skipping initialization."
                touch /data/mysql_tables_initialized
              else
                echo "Table '${TARGET_TABLE}' not found. Database structure will be initialized by the main container."
              fi
              # --------------------------------------------------------------------------
              # 脚本结束
              # --------------------------------------------------------------------------

								`,
							},
							Env: []corev1.EnvVar{
								{
									Name: "MYSQL_HOST",
									ValueFrom: &corev1.EnvVarSource{
										ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "blog-config",
											},
											Key: "mysql.host",
										},
									},
								},
								{
									Name: "TARGET_DB",
									ValueFrom: &corev1.EnvVarSource{
										ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "blog-config",
											},
											Key: "database.name",
										},
									},
								},
								{
									Name: "MYSQL_PORT",
									ValueFrom: &corev1.EnvVarSource{
										ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "blog-config",
											},
											Key: "mysql.port",
										},
									},
								},
								{
									Name:  "TARGET_TABLE",
									Value: "users",
								},
								{
									Name:  "MYSQL_ROOT_PASSWORD",
									Value: "123456abc",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "persistent-storage",
									MountPath: "/data",
								},
							},
						},
						{
							Name: "check-es",
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceCPU: resource.Quantity{
										Format: "500m",
									},
									corev1.ResourceMemory: resource.Quantity{
										Format: "256Mi",
									},
								},
								Requests: corev1.ResourceList{
									corev1.ResourceCPU: resource.Quantity{
										Format: "100M",
									},
									corev1.ResourceMemory: resource.Quantity{
										Format: "128Mi",
									},
								},
							},
							Image: "curlimages/curl",
							Command: []string{
								"sh",
								"-c",
								`until curl -sSfk -u "elastic:$(ES_PASSWORD)" https://$(ES_HOST):$(ES_PORT); do
                echo "Waiting for Elasticsearch...";
                sleep 2;
              done
								`,
							},
							Env: []corev1.EnvVar{
								{
									Name: "ES_HOST",
									ValueFrom: &corev1.EnvVarSource{
										ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "blog-config",
											},
											Key: "es.host",
										},
									},
								},
								{
									Name: "ES_PORT",
									ValueFrom: &corev1.EnvVarSource{
										ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "blog-config",
											},
											Key: "es.port",
										},
									},
								},
								{
									Name: "ES_INDEX",
									ValueFrom: &corev1.EnvVarSource{
										ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "blog-config",
											},
											Key: "es.index",
										},
									},
								},
								{
									Name: "ES_PASSWORD",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "es-cluster-es-elastic-user",
											},
											Key: "elastic",
										},
									},
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "persistent-storage",
									MountPath: "/data",
								},
							},
						},
						{
							Name: "check-redis",
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceCPU: resource.Quantity{
										Format: "500m",
									},
									corev1.ResourceMemory: resource.Quantity{
										Format: "256Mi",
									},
								},
								Requests: corev1.ResourceList{
									corev1.ResourceCPU: resource.Quantity{
										Format: "100M",
									},
									corev1.ResourceMemory: resource.Quantity{
										Format: "128Mi",
									},
								},
							},
							Image: "busybox",
							Command: []string{
								"sh",
								"-c",
								`until nc -z $(REDIS_HOST) $(REDIS_PORT); do
                echo "Waiting for Redis...";
                sleep 2;
              done
								`,
							},
							Env: []corev1.EnvVar{
								{
									Name: "REDIS_HOST",
									ValueFrom: &corev1.EnvVarSource{
										ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "blog-config",
											},
											Key: "redis.host",
										},
									},
								},
								{
									Name: "REDIS_PORT",
									ValueFrom: &corev1.EnvVarSource{
										ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "blog-config",
											},
											Key: "redis.port",
										},
									},
								},
							},
						},
					},
					ImagePullSecrets: []corev1.LocalObjectReference{{Name: blog.Spec.ImagePullSecret}},
					Containers: []corev1.Container{{
						Name:  "backend",
						Image: blog.Spec.BackendImage,
						Ports: []corev1.ContainerPort{{ContainerPort: 8080}},
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU: resource.Quantity{
									Format: "2",
								},
								corev1.ResourceMemory: resource.Quantity{
									Format: "1Gi",
								},
							},
							Requests: corev1.ResourceList{
								corev1.ResourceCPU: resource.Quantity{
									Format: "1",
								},
								corev1.ResourceMemory: resource.Quantity{
									Format: "512Mi",
								},
							},
						},
						Command: com,
						Env: []corev1.EnvVar{
							{
								Name:  "ES_CACERT_PATH",
								Value: "/es-certs/ca.crt",
							},
						},
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "persistent-storage",
								MountPath: "/data",
							},
							{
								Name:      "config-volume",
								MountPath: "/root/config.yaml",
								SubPath:   "config.yaml",
							},
							{
								Name:      "es-certs",
								MountPath: "/es-certs",
								ReadOnly:  true,
							},
						},
						ReadinessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								HTTPGet: &corev1.HTTPGetAction{
									Path: "/healthz",
									Port: intstr.FromInt(8080),
								},
							},
							InitialDelaySeconds: 10,
							TimeoutSeconds:      5,
						},
						LivenessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								TCPSocket: &corev1.TCPSocketAction{
									Port: intstr.FromInt(8080),
								},
							},
							InitialDelaySeconds: 10,
							TimeoutSeconds:      5,
						},
					}},
					Volumes: []corev1.Volume{
						{
							Name: "persistent-storage",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: "blog-storage",
								},
							},
						},
						{
							Name: "es-certs",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "es-cluster-es-http-certs-internal",
									Items: []corev1.KeyToPath{
										{
											Key:  "ca.crt",
											Path: "ca.crt",
										},
									},
								},
							},
						},
						{
							Name: "config-volume",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "blog-config",
									},
									Items: []corev1.KeyToPath{
										{
											Key:  "config.yaml",
											Path: "config.yaml",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	ctrl.SetControllerReference(blog, dep, r.Scheme)
	return dep
}

// --- Frontend Resources ---

func (r *BlogReconciler) buildFrontendConfigMap(blog *blogv1alpha1.Blog) *corev1.ConfigMap {
	// ...
	var configYaml string = `
log_format main_ext '[$time_local] "$request" $status $body_bytes_sent '
                      '"$http_referer" "$http_user_agent" '
                      'real_ip:"$http_x_forwarded_for" proxy_ip:"$remote_addr"'
                      'client_ip:"$http_x_client_ip"'
                      'ipcountry:"$http_ipcountry"';
    server {
      listen 80;
      server_name localhost;

      access_log /var/log/nginx/access.log main_ext;

      location / {
          root /usr/share/nginx/html;
          try_files $uri $uri/ /index.html;
      }

      location /api/ {
          proxy_pass http://blog-backend:8080/api/;
          proxy_set_header Host $host;
          proxy_set_header X-Real-IP $remote_addr;
          proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
          proxy_set_header X-Forwarded-Proto $scheme;
      }
    }

`
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "blog-frontend-nginx-config",
			Namespace: blog.Namespace,
		},
		Data: map[string]string{
			"config.conf": configYaml,
		},
	}
	ctrl.SetControllerReference(blog, cm, r.Scheme)
	return cm
}

func (r *BlogReconciler) buildFrontendDeployment(blog *blogv1alpha1.Blog, image, version, name1 string) *appsv1.Deployment {
	replicas := int32(1)
	name := "blog-frontend-" + name1
	labels := map[string]string{
		"app":     "blog-frontend",
		"version": version,
	}

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: blog.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "frontend",
						Image: image,
						Ports: []corev1.ContainerPort{{ContainerPort: 80}},
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU: resource.Quantity{
									Format: "200M",
								},
								corev1.ResourceMemory: resource.Quantity{
									Format: "512Mi",
								},
							},
							Requests: corev1.ResourceList{
								corev1.ResourceCPU: resource.Quantity{
									Format: "100M",
								},
								corev1.ResourceMemory: resource.Quantity{
									Format: "256Mi",
								},
							},
						},
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "nginx-config-volume",
								MountPath: " /etc/nginx/conf.d/",
							},
						},
						ReadinessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								HTTPGet: &corev1.HTTPGetAction{
									Path: "/",
									Port: intstr.FromInt(80),
								},
							},
							InitialDelaySeconds: 5,
							PeriodSeconds:       10,
						},
						LivenessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								HTTPGet: &corev1.HTTPGetAction{
									Path: "/",
									Port: intstr.FromInt(80),
								},
							},
							InitialDelaySeconds: 15,
							PeriodSeconds:       20,
						},
					}},
					Volumes: []corev1.Volume{
						{
							Name: "nginx-config-volume",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "blog-frontend-nginx-config",
									},
								},
							},
						},
					},
				},
			},
		},
	}
	ctrl.SetControllerReference(blog, dep, r.Scheme)
	return dep
}

func (r *BlogReconciler) buildFrontendService(blog *blogv1alpha1.Blog, version, name1 string) *corev1.Service {

	name := "blog-frontend-" + name1
	labels := map[string]string{
		"app":     "blog-frontend",
		"version": version,
	}
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: blog.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports:    []corev1.ServicePort{{Port: 80, TargetPort: intstr.FromInt(80)}},
		},
	}
	ctrl.SetControllerReference(blog, svc, r.Scheme)
	return svc
}

// --- Network Resources ---

func (r *BlogReconciler) buildGateway(blog *blogv1alpha1.Blog) *gatewayv1.Gateway {
	hostname := gatewayv1.Hostname(blog.Spec.Domain)
	allNamespaces := gatewayv1.NamespacesFromAll
	terminate := gatewayv1.TLSModeTerminate
	gw := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "blog",
			Namespace: "istio-system",
		},
		Spec: gatewayv1.GatewaySpec{
			GatewayClassName: "istio",
			Listeners: []gatewayv1.Listener{
				{
					Name:     "http",
					Hostname: &hostname,
					Port:     80,
					Protocol: gatewayv1.HTTPProtocolType,
					AllowedRoutes: &gatewayv1.AllowedRoutes{
						Namespaces: &gatewayv1.RouteNamespaces{
							From: &allNamespaces,
						},
					},
				},
				{
					Name:     "https",
					Hostname: &hostname,
					Port:     443,
					Protocol: gatewayv1.HTTPSProtocolType,
					TLS: &gatewayv1.GatewayTLSConfig{
						Mode: &terminate,
						CertificateRefs: []gatewayv1.SecretObjectReference{
							{
								Name: gatewayv1.ObjectName(blog.Spec.TLSSecretName),
							},
						},
					},
					AllowedRoutes: &gatewayv1.AllowedRoutes{
						Namespaces: &gatewayv1.RouteNamespaces{
							From: &allNamespaces,
						},
					},
				},
			},
		},
	}
	// **移除跨命名空间的 OwnerReference**
	//ctrl.SetControllerReference(blog, gw, r.Scheme)
	return gw
}

func (r *BlogReconciler) buildHTTPRoute(blog *blogv1alpha1.Blog) *gatewayv1.HTTPRoute {

	// 这里可以加入逻辑，如果 canary image 存在，就创建带 canary 规则的 route
	var namespace gatewayv1.Namespace = "istio-system"
	pf := gatewayv1.PathMatchPathPrefix
	me := gatewayv1.HeaderMatchExact
	//var v string = "/"
	hr := &gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "blog-http",
			Namespace: blog.Namespace,
		},
		Spec: gatewayv1.HTTPRouteSpec{
			CommonRouteSpec: gatewayv1.CommonRouteSpec{
				ParentRefs: []gatewayv1.ParentReference{
					{
						Name:      "blog",
						Namespace: &namespace,
					},
				},
			},
			Hostnames: []gatewayv1.Hostname{
				"blog.678645.xyz",
			},
			Rules: []gatewayv1.HTTPRouteRule{
				{
					Matches: []gatewayv1.HTTPRouteMatch{
						{
							Path: &gatewayv1.HTTPPathMatch{
								Type:  &pf,
								Value: pointer.String("/"),
								//Value: &v,
							},
							Headers: []gatewayv1.HTTPHeaderMatch{
								{
									Type:  &me,
									Name:  "ipcountry",
									Value: "CN",
								},
							},
						},
					},
					BackendRefs: []gatewayv1.HTTPBackendRef{
						{
							BackendRef: gatewayv1.BackendRef{
								BackendObjectReference: gatewayv1.BackendObjectReference{
									Name: "blog-frontend-v1-0-1",
									Port: (*gatewayv1.PortNumber)(pointer.Int32(80)),
								},
							},
						},
					},
				},
				{
					Matches: []gatewayv1.HTTPRouteMatch{
						{
							Path: &gatewayv1.HTTPPathMatch{
								Type:  &pf,
								Value: pointer.String("/"),
								//Value: &v,
							},
						},
					},
					BackendRefs: []gatewayv1.HTTPBackendRef{
						{
							BackendRef: gatewayv1.BackendRef{
								BackendObjectReference: gatewayv1.BackendObjectReference{
									Name: "blog-frontend-v2-0-1",
									Port: (*gatewayv1.PortNumber)(pointer.Int32(80)),
								},
							},
						},
					},
				},
			},
		},
	}
	ctrl.SetControllerReference(blog, hr, r.Scheme)
	return hr
}
