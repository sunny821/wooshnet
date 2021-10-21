package cniapi

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/mdlayher/vsock"

	networkv1 "wooshnet/apis/network/v1"
	"wooshnet/pkg/httpclient"
	"wooshnet/pkg/server"
	"wooshnet/pkg/wooshtools"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"
)

var ServerSocket string = "/var/run/wooshnet/woosh.sock"
var VSockPort uint = 777

var printOnce sync.Once

func checkServer(cli *httpclient.HttpClient) error {
	var server string
	defer printOnce.Do(func() {
		klog.Infoln("server:", server)
	})

	if !wooshtools.FileExist(ServerSocket) {
		conn, err := vsock.Dial(vsock.Host, cli.Port)
		if err != nil {
			// klog.Errorln(err)
		} else {
			server = conn.LocalAddr().String() + " -> " + conn.RemoteAddr().String()
			conn.Close()
			return nil
		}
		conn, err = vsock.Dial(vsock.Hypervisor, cli.Port)
		if err != nil {
			// klog.Errorln(err)
		} else {
			server = conn.LocalAddr().String() + " -> " + conn.RemoteAddr().String()
			conn.Close()
			return nil
		}
		return fmt.Errorf("%v not found", ServerSocket)
	} else {
		server = ServerSocket
	}

	return nil
}

func NewServerClient() *httpclient.HttpClient {
	httpCli := httpclient.HttpClient{
		Address: ServerSocket,
		Mode:    "unix",
	}
	if !wooshtools.FileExist(ServerSocket) {
		httpCli.Mode = "vsock"
		httpCli.ContextID = vsock.Hypervisor
		httpCli.Port = uint32(VSockPort)
	}

	return &httpCli
}

func GetWooshPortFromPod(nsname, podname string) (*networkv1.WooshPort, error) {
	if len(podname) == 0 {
		err := errors.NewBadRequest("podname是空的")
		return nil, err
	}
	if len(nsname) == 0 {
		err := errors.NewBadRequest("namespace是空的")
		return nil, err
	}
	httpCli := NewServerClient()
	var rsp server.WooshPortResponse
	statuscode, err := httpCli.HttpGetJson("/Kubernetes/WooshPort/"+nsname+"/"+podname, &rsp)
	if err != nil {
		return nil, err
	}
	if statuscode != http.StatusOK {
		if rsp.Error != "" {
			err = fmt.Errorf(rsp.Error)
		} else {
			err = fmt.Errorf(http.StatusText(statuscode))
		}
		return nil, err
	}

	return &rsp.WooshPort, nil
}

func GetWooshPortFromCR(nsname, wpname string) (*networkv1.WooshPort, error) {
	if len(wpname) == 0 {
		err := errors.NewBadRequest("wpname是空的")
		return nil, err
	}
	if len(nsname) == 0 {
		err := errors.NewBadRequest("namespace是空的")
		return nil, err
	}
	httpCli := NewServerClient()
	var rsp server.WooshPortResponse
	statuscode, err := httpCli.HttpGetJson("/Kubernetes/WooshPortCR/"+nsname+"/"+wpname, &rsp)
	if err != nil {
		return nil, err
	}
	if statuscode != http.StatusOK {
		if rsp.Error != "" {
			err = fmt.Errorf(rsp.Error)
		} else {
			err = fmt.Errorf(http.StatusText(statuscode))
		}
		return nil, err
	}

	return &rsp.WooshPort, nil
}

func DeleteWooshPortCR(nsname, wpname string) error {
	if len(wpname) == 0 {
		err := errors.NewBadRequest("wpname是空的")
		return err
	}
	if len(nsname) == 0 {
		err := errors.NewBadRequest("namespace是空的")
		return err
	}
	httpCli := NewServerClient()
	var rsp server.WooshPortResponse
	statuscode, err := httpCli.HttpDeleteJson("/Kubernetes/WooshPortCR/"+nsname+"/"+wpname, nil, &rsp)
	if err != nil {
		return err
	}
	if statuscode != http.StatusOK {
		if rsp.Error != "" {
			err = fmt.Errorf(rsp.Error)
		} else {
			err = fmt.Errorf(http.StatusText(statuscode))
		}
		return err
	}

	return nil
}

func CreateWooshPortCR(req *server.WooshPortRequest) error {
	httpCli := NewServerClient()
	var rsp server.WooshPortResponse
	statuscode, err := httpCli.HttpPostJson("/Kubernetes/WooshPortCR/"+req.Namespace+"/"+req.Name, req, &rsp)
	if err != nil {
		return err
	}
	if statuscode != http.StatusOK {
		if rsp.Error != "" {
			err = fmt.Errorf(rsp.Error)
		} else {
			err = fmt.Errorf(http.StatusText(statuscode))
		}
		return err
	}

	return nil
}

func UpdateDeviceReady(nsname, wpname string) error {
	if len(wpname) == 0 {
		err := errors.NewBadRequest("podname是空的")
		return err
	}
	if len(nsname) == 0 {
		err := errors.NewBadRequest("namespace是空的")
		return err
	}
	httpCli := NewServerClient()
	var rsp server.WooshPortResponse
	statuscode, err := httpCli.HttpPatchJson("/Kubernetes/WooshPort.DeviceReady/"+nsname+"/"+wpname, nil, &rsp)
	if err != nil {
		return err
	}
	if statuscode != http.StatusOK {
		if rsp.Error != "" {
			err = fmt.Errorf(rsp.Error)
		} else {
			err = fmt.Errorf(http.StatusText(statuscode))
		}
		return err
	}

	return nil
}
