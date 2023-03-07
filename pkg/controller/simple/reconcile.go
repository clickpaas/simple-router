package simple

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	apiappsv1 "k8s.io/api/apps/v1"
	apicorev1 "k8s.io/api/core/v1"
	k8serror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

const (
	APPLICATION_TAG = "APPLICATION_TAG"
)

func (c *controller) Reconcile(key string) error {
	// if deployment ,所有进来的deployment都是以dmo开头的
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(err)
		return nil
	}

	// service'name that not containts any tag and deployment are the same
	svcName, err := c.deploymentLister.Deployments(namespace).Get(name)
	if k8serror.IsNotFound(err) {
		// resource is not existed, then return do nothing, no reconcile any more
		utilruntime.HandleError(err)
		return nil
	}
	if err != nil {
		return fmt.Errorf("get deployment failed %v:%v", name, err)
	}
	// 无差别的处理每一个deployment 用来补充globalTags， 这里不做任何区别
	c.fetchTagsFromDeploymentSpec(svcName)

	if c.serviceHasTaged(svcName) {
		// service 以tag结尾的。
		return nil
	}
	return c.serviceSync(svcName)
}

// 判断deploy是否可以继续执行service 同步处理, 只有下面两种方式可以继续处理
// 1. deploy如果在白名单里面可以继续处理
// 2. 如果deployment不在白名单里面，那么不带tag结尾的也可以接续处理
func (c *controller) serviceHasTaged(dpl *apiappsv1.Deployment) bool {
	if strings.Contains(dpl.GetName(), "-tag") {
		return true
	}
	return false
}

// serviceSync
func (c *controller) serviceSync(dpl *apiappsv1.Deployment) error {
	//svc,err := c.serviceLister.List(labels.SelectorFromSet(dpl.Labels))
	svc, err := c.serviceLister.Services(dpl.Namespace).Get(deploymentToServiceName(dpl.GetName()))
	if k8serror.IsNotFound(err) {
		utilruntime.HandleError(err)
		return nil
	}
	if err != nil {
		return fmt.Errorf("get service failed: %v", err)
	}
	// 处理globales
	// level1 存在 level2 存在不做处理
	// level1 不存在， level2不存在， level1, leve2 都指向default
	// level1 存在

	hasDealedTags := make(map[string]struct{})
	klog.Errorf("debug globalTags  %v", c.globalTags)
	for _, tag := range c.globalTags {
		// 第一级tag 是否存在
		fmt.Fprintf(os.Stdout, "%v  %v", tag.firstLevel, tag.secondLevel)
		firstLevelExisted := true
		hasDealedTags[tag.firstLevel] = struct{}{}
		hasDealedTags[tag.secondLevel] = struct{}{}
		flsvc, err := c.serviceLister.Services(svc.GetNamespace()).Get(serviceNameWithTags(svc.GetName(), tag.firstLevel))
		if k8serror.IsNotFound(err) {
			// 第一级不存在，则指向默认的
			firstLevelExisted = false
			flsvc, err = c.kubeClientSet.CoreV1().Services(svc.GetNamespace()).
				Create(context.TODO(), buildExternalNameService(serviceNameWithTags(svc.GetName(), tag.firstLevel), svc), metav1.CreateOptions{})
		}
		if err != nil {
			return fmt.Errorf("create svc if not existed: %v", err)
		}
		_, err = c.serviceLister.Services(dpl.GetNamespace()).Get(serviceNameWithTags(svc.GetName(), tag.secondLevel))
		if k8serror.IsNotFound(err) {
			if firstLevelExisted {
				// 第二级不存在，但是第一级存在，则指向第一级
				_, err = c.kubeClientSet.CoreV1().Services(svc.GetNamespace()).
					Create(context.TODO(), buildExternalNameService(serviceNameWithTags(svc.GetName(), tag.secondLevel), flsvc), metav1.CreateOptions{})
			} else {
				// 第二级别不存在，但是第一级存在，则指向默认级别
				_, err = c.kubeClientSet.CoreV1().Services(svc.GetNamespace()).
					Create(context.TODO(), buildExternalNameService(serviceNameWithTags(svc.GetName(), tag.secondLevel), svc), metav1.CreateOptions{})
			}
		}
		if err != nil {
			return fmt.Errorf("hanlder mult tag service err: %v", err)
		}
	}
	// 处理其他的, tag没有0的
	for i := 1; i <= c.config.NumTags; i++ {
		tag := "tag" + strconv.Itoa(i)
		if _, ok := hasDealedTags[tag]; ok {
			// 已经存在对应的service
			continue
		}
		svcTagName := serviceNameWithTags(svc.GetName(), tag)
		_, err = c.serviceLister.Services(svc.GetNamespace()).Get(svcTagName)
		if k8serror.IsNotFound(err) {
			_, err = c.kubeClientSet.CoreV1().Services(svc.GetNamespace()).
				Create(context.TODO(), buildExternalNameService(serviceNameWithTags(svc.GetName(), tag), svc), metav1.CreateOptions{})
		}
		if err != nil {
			return fmt.Errorf("create service if not existed,err : %v", err)
		}
	}

	return nil
}

// fetchTagsFromDeploymentSpec 从 deployment environment 里面提取tags， 如果tags 解析后个数> 2 ，则报错返回，并且不在做任何处理
func (c *controller) fetchTagsFromDeploymentSpec(dpl *apiappsv1.Deployment) {
	// 所有dmo-xxxx-rest-xxx的deployment里面有tag 都提取出来
	dplTags := dpl.Spec.Template.Spec.Containers[0].Env
	var tagValue string
	for _, env := range dplTags {
		if env.Name == APPLICATION_TAG {
			tagValue = env.Value
		}
	}
	if tagValue == "" {
		return
	}
	tmpTagSplit := strings.Split(tagValue, "|")
	if len(tmpTagSplit) > 2 {
		utilruntime.HandleError(fmt.Errorf("Invalid Tags %s: Tags to many %v", dpl.GetName(), tagValue))
	}
	if _, ok := c.globalTags[tmpTagSplit[0]+"|"+tmpTagSplit[1]]; !ok {
		c.globalTags[tmpTagSplit[0]+"|"+tmpTagSplit[1]] = tagItem{
			firstLevel:  tmpTagSplit[0],
			secondLevel: tmpTagSplit[1],
		}
	}
}

func buildExternalNameService(exterSvcName string, targetSvc *apicorev1.Service) *apicorev1.Service {
	svc := &apicorev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: targetSvc.Namespace,
			Name:      exterSvcName,
		},
		Spec: apicorev1.ServiceSpec{
			Type:         apicorev1.ServiceTypeExternalName,
			ExternalName: fmt.Sprintf("%s.%s.svc.cluster.local", targetSvc.GetName(), targetSvc.GetNamespace()),
		},
	}
	return svc
}

func deploymentToServiceName(dplName string) string {
	return strings.TrimSuffix(dplName, "-deploy")
}

func serviceNameWithTags(svcName, tag string) string {
	return svcName + "-" + tag
}
