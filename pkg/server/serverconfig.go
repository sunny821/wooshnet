package server

import (
	"context"
	"fmt"
	"net/http"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	networkv1 "wooshnet/apis/network/v1"

	corev1 "k8s.io/api/core/v1"

	networkcontrollers "wooshnet/controllers/network"

	"github.com/gin-gonic/gin"
)

type ServerConfig struct {
	K8sCli client.Client
}

func (c *ServerConfig) Register(eng *gin.Engine) {
	eng.GET("/Kubernetes/WooshPort/:namespace/:podname", c.getWooshPort)
	eng.GET("/Kubernetes/WooshPortCR/:namespace/:wpname", c.getWooshPortCR)
	eng.DELETE("/Kubernetes/WooshPortCR/:namespace/:wpname", c.deleteWooshPortCR)
	eng.POST("/Kubernetes/WooshPortCR/:namespace/:wpname", c.createWooshPortCR)
	eng.PATCH("/Kubernetes/WooshPort.DeviceReady/:namespace/:wpname", c.updateDeviceReady)
}

type Response struct {
	Error string `json:"error"`
}

type WooshPortRequest struct {
	networkv1.WooshPort
}

type WooshPortResponse struct {
	Response
	WooshPort networkv1.WooshPort `json:"wooshPort,omitempty"`
}

func (c *ServerConfig) getWooshPort(r *gin.Context) {
	namespace := r.Param("namespace")
	podname := r.Param("podname")
	var rsp WooshPortResponse
	key := types.NamespacedName{
		Namespace: namespace,
		Name:      podname,
	}
	pod := &corev1.Pod{}
	err := c.K8sCli.Get(context.Background(), key, pod)
	if err != nil {
		rsp.Error = fmt.Sprintf("%v", err)
		r.JSONP(http.StatusInternalServerError, rsp)
		return
	}
	wpname := podname
	annos := pod.Annotations
	if annos == nil {
		annos = make(map[string]string)
	}
	if annos[networkcontrollers.WooshPortName] != "" {
		wpname = annos[networkcontrollers.WooshPortName]
	}
	nsd := types.NamespacedName{
		Name:      wpname,
		Namespace: namespace,
	}
	var wp networkv1.WooshPort
	err = c.K8sCli.Get(context.Background(), nsd, &wp)
	if err != nil {
		rsp.Error = fmt.Sprintf("%v", err)
		r.JSONP(http.StatusInternalServerError, rsp)
	} else {
		rsp.WooshPort = wp
		r.JSONP(http.StatusOK, rsp)
	}
}

func (c *ServerConfig) getWooshPortCR(r *gin.Context) {
	namespace := r.Param("namespace")
	wpname := r.Param("wpname")

	var rsp WooshPortResponse
	nsd := types.NamespacedName{
		Namespace: namespace,
		Name:      wpname,
	}
	var wp networkv1.WooshPort
	err := c.K8sCli.Get(context.Background(), nsd, &wp)
	if err != nil {
		rsp.Error = fmt.Sprintf("%v", err)
		r.JSONP(http.StatusInternalServerError, rsp)
	} else {
		rsp.WooshPort = wp
		r.JSONP(http.StatusOK, rsp)
	}
}

func (c *ServerConfig) deleteWooshPortCR(r *gin.Context) {
	namespace := r.Param("namespace")
	wpname := r.Param("wpname")

	var rsp Response
	nsd := types.NamespacedName{
		Namespace: namespace,
		Name:      wpname,
	}
	var wp networkv1.WooshPort
	err := c.K8sCli.Get(context.Background(), nsd, &wp)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			rsp.Error = fmt.Sprintf("%v", err)
			r.JSONP(http.StatusInternalServerError, rsp)
		} else {
			r.JSONP(http.StatusOK, rsp)
		}
	} else {
		wp.Spec.Deleted = true
		err = c.K8sCli.Update(context.TODO(), &wp)
		if err != nil {
			rsp.Error = fmt.Sprintf("%v", err)
			r.JSONP(http.StatusInternalServerError, rsp)
		} else {
			r.JSONP(http.StatusOK, rsp)
		}
	}
}

func (c *ServerConfig) createWooshPortCR(r *gin.Context) {
	// namespace := r.Param("namespace")
	// wpname := r.Param("wpname")

	var req WooshPortRequest
	err := r.BindJSON(&req)
	if err != nil {
		r.JSONP(http.StatusInternalServerError, err)
		return
	}
	var rsp Response
	err = c.K8sCli.Create(context.Background(), &req.WooshPort)
	if err != nil {
		rsp.Error = err.Error()
		r.JSONP(http.StatusInternalServerError, rsp)
	} else {
		r.JSONP(http.StatusOK, rsp)
	}
	return
}

func (c *ServerConfig) updateDeviceReady(r *gin.Context) {
	namespace := r.Param("namespace")
	wpname := r.Param("wpname")

	var rsp Response
	nsd := types.NamespacedName{
		Name:      wpname,
		Namespace: namespace,
	}
	var wp networkv1.WooshPort
	err := c.K8sCli.Get(context.Background(), nsd, &wp)
	if err != nil {
		rsp.Error = fmt.Sprintf("%v", err)
		r.JSONP(http.StatusInternalServerError, rsp)
		return
	}
	wp.Status.DeviceReady = false
	wp.Status.PodReady = false
	err = c.K8sCli.Status().Update(context.Background(), &wp)
	if err != nil {
		rsp.Error = fmt.Sprintf("%v", err)
		r.JSONP(http.StatusInternalServerError, rsp)
	} else {
		r.JSONP(http.StatusOK, rsp)
	}
}
