package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/mdlayher/vsock"
	"k8s.io/klog/v2"
)

type HttpClient struct {
	Mode       string
	Address    string
	ContextID  uint32
	Host       string
	Port       uint32
	URL        string
	Addrs      []string
	Index      int
	Once       sync.Once
	Transports map[string]*http.Transport
}

func (h *HttpClient) Init() {
	h.Once.Do(func() {
		if len(h.Mode) == 0 {
			h.Mode = "unix"
		}
		switch h.Mode {
		case "unix", "UNIX", "Unix":
			h.Addrs = strings.Split(h.Address, ",")
			h.Index = h.Index % len(h.Addrs)
			h.URL = "http://dummy"
		case "vsock", "VSOCK", "Vsock":
			h.Addrs = append(h.Addrs, fmt.Sprintf("%v:%v", h.ContextID, h.Port))
			h.Index = h.Index % len(h.Addrs)
			h.URL = "http://dummy"
		case "tcp", "TCP", "Tcp":
			h.Addrs = strings.Split(h.Address, ",")
			h.Index = h.Index % len(h.Addrs)
			h.Host = h.Addrs[h.Index]
			h.URL = "http://" + h.Host
		case "udp", "UDP", "Udp":
			h.Addrs = strings.Split(h.Address, ",")
			h.Index = h.Index % len(h.Addrs)
			h.Host = h.Addrs[h.Index]
			h.URL = "http://" + h.Host
		default:
			h.Mode = "unix"
			h.Addrs = strings.Split(h.Address, ",")
			h.Index = h.Index % len(h.Addrs)
			h.URL = "http://dummy"
		}
		h.Transports = make(map[string]*http.Transport, len(h.Addrs))
		for _, addr := range h.Addrs {
			h.Transports[addr] = &http.Transport{DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				switch h.Mode {
				case "unix", "UNIX", "Unix":
					return net.Dial("unix", addr)
				case "vsock", "VSOCK", "Vsock":
					return vsock.Dial(h.ContextID, h.Port)
				case "tcp", "TCP", "Tcp":
					return net.Dial("tcp", addr)
				case "udp", "UDP", "Udp":
					return net.Dial("udp", addr)
				default:
					h.Mode = "unix"
				}
				return net.Dial("unix", addr)
			}}
		}
	})
}

func (h *HttpClient) NewClient() (string, *http.Client) {
	h.Init()
	h.Index = (h.Index + 1) % len(h.Addrs)
	h.Host = h.Addrs[h.Index]
	client := http.DefaultClient
	client.Timeout = time.Second * 60
	client.Transport = h.Transports[h.Host]
	switch h.Mode {
	case "unix", "UNIX", "Unix":
		h.URL = "http://dummy"
	case "vsock", "VSOCK", "Vsock":
		h.URL = "http://dummy"
	case "tcp", "TCP", "Tcp":
		h.URL = "http://" + h.Host
	case "udp", "UDP", "Udp":
		h.URL = "http://" + h.Host
	default:
		h.Mode = "unix"
		h.URL = "http://dummy"
	}
	return h.URL, client
}

func (h *HttpClient) Close() {
	for _, trans := range h.Transports {
		if trans != nil {
			trans.CloseIdleConnections()
		}
	}
}

func (h *HttpClient) HttpGet(uri string) (int, []byte, error) {
	h.Init()
	url, client := h.NewClient()
	defer client.CloseIdleConnections()
	url = url + uri
	// klog.Infoln("GET", url)
	rsp, err := client.Get(url)
	if err != nil {
		// klog.Errorf("%#v, %v", rsp, err)
		return http.StatusNotFound, []byte{}, err
	}
	// klog.Infof("%v", rsp)
	buf, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		klog.Errorf("%v", err)
		return rsp.StatusCode, []byte{}, err
	}
	// klog.Infof("%s", buf)
	return rsp.StatusCode, buf, nil
}

func (h *HttpClient) HttpPut(uri, contentType string, body []byte) (int, []byte, error) {
	h.Init()
	url, client := h.NewClient()
	defer client.CloseIdleConnections()
	url = url + uri
	// klog.Infoln("PUT", url)
	req, err := http.NewRequest("PUT", url, bytes.NewReader(body))
	if err != nil {
		return http.StatusNotFound, []byte{}, err
	}
	req.Header.Set("Content-Type", contentType)
	rsp, err := client.Do(req)
	if err != nil {
		klog.Errorf("%#v, %v", rsp, err)
		return http.StatusNotFound, []byte{}, err
	}
	// klog.Infof("%v", rsp)
	buf, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		klog.Errorf("%v", err)
		return rsp.StatusCode, []byte{}, err
	}
	// klog.Infof("%s", buf)
	return rsp.StatusCode, buf, nil
}

func (h *HttpClient) HttpPost(uri, contentType string, body []byte) (int, []byte, error) {
	h.Init()
	url, client := h.NewClient()
	defer client.CloseIdleConnections()
	url = url + uri
	// klog.Infoln("POST", url)
	rsp, err := client.Post(url, contentType, bytes.NewReader(body))
	if err != nil {
		klog.Errorf("%#v, %v", rsp, err)
		return http.StatusNotFound, []byte{}, err
	}
	// klog.Infof("%v", rsp)
	buf, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		klog.Errorf("%v", err)
		return rsp.StatusCode, []byte{}, err
	}
	// klog.Infof("%s", buf)
	return rsp.StatusCode, buf, nil
}

func (h *HttpClient) HttpPatch(uri, contentType string, body []byte) (int, []byte, error) {
	h.Init()
	url, client := h.NewClient()
	defer client.CloseIdleConnections()
	url = url + uri
	// klog.Infoln(http.MethodPatch, url)
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	rsp, err := client.Do(req)
	if err != nil {
		klog.Errorf("%#v, %v", rsp, err)
		return http.StatusNotFound, []byte{}, err
	}
	buf, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		klog.Errorf("%v", err)
		return rsp.StatusCode, []byte{}, err
	}
	// klog.Infof("%s", buf)
	return rsp.StatusCode, buf, nil
}

func (h *HttpClient) HttpDelete(uri, contentType string, body []byte) (int, []byte, error) {
	h.Init()
	url, client := h.NewClient()
	defer client.CloseIdleConnections()
	url = url + uri
	// klog.Infoln("DELETE", url)
	req, err := http.NewRequest("DELETE", url, bytes.NewReader(body))
	if err != nil {
		return http.StatusBadRequest, nil, err
	}
	req.Header.Set("Content-Type", contentType)
	rsp, err := client.Do(req)
	if err != nil {
		klog.Errorf("%#v, %v", rsp, err)
		return http.StatusNotFound, []byte{}, err
	}
	// klog.Infof("%v", rsp)
	buf, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		klog.Errorf("%v", err)
		return rsp.StatusCode, []byte{}, err
	}
	// klog.Infof("%s", buf)
	return rsp.StatusCode, buf, nil
}

func (h *HttpClient) HttpGetJson(uri string, result interface{}) (int, error) {
	statuscode, buf, err := h.HttpGet(uri)
	if err != nil {
		return statuscode, err
	}

	if len(buf) > 0 {
		err = json.Unmarshal(buf, &result)
		if err != nil {
			return statuscode, fmt.Errorf("%s", buf)
		}
	}

	return statuscode, nil
}

func (h *HttpClient) HttpPutJson(uri string, req interface{}, result interface{}) (int, error) {
	var body []byte
	var err error
	if req != nil {
		body, err = json.Marshal(req)
		if err != nil {
			return http.StatusBadRequest, err
		}
	}
	contentType := "application/json"
	statuscode, buf, err := h.HttpPut(uri, contentType, body)
	if err != nil {
		klog.Errorln(err)
		return statuscode, err
	}

	if len(buf) > 0 {
		err = json.Unmarshal(buf, &result)
		if err != nil {
			return statuscode, fmt.Errorf("%s", buf)
		}
	}

	return statuscode, nil
}

func (h *HttpClient) HttpPostJson(uri string, req interface{}, result interface{}) (int, error) {
	var body []byte
	var err error
	if req != nil {
		body, err = json.Marshal(req)
		if err != nil {
			return http.StatusBadRequest, err
		}
	}
	contentType := "application/json"
	statuscode, buf, err := h.HttpPost(uri, contentType, body)
	if err != nil {
		klog.Errorln(err)
		return statuscode, err
	}

	if len(buf) > 0 {
		err = json.Unmarshal(buf, &result)
		if err != nil {
			return statuscode, fmt.Errorf("%s", buf)
		}
	}

	return statuscode, nil
}

func (h *HttpClient) HttpPatchJson(uri string, req interface{}, result interface{}) (int, error) {
	var body []byte
	var err error
	if req != nil {
		body, err = json.Marshal(req)
		if err != nil {
			return http.StatusBadRequest, err
		}
	}
	contentType := "application/json"
	statuscode, buf, err := h.HttpPatch(uri, contentType, body)
	if err != nil {
		klog.Errorln(err)
		return statuscode, err
	}

	if len(buf) > 0 {
		err = json.Unmarshal(buf, &result)
		if err != nil {
			return statuscode, fmt.Errorf("%s", buf)
		}
	}

	return statuscode, nil
}

func (h *HttpClient) HttpDeleteJson(uri string, req interface{}, result interface{}) (int, error) {
	var body []byte
	var err error
	if req != nil {
		body, err = json.Marshal(req)
		if err != nil {
			return http.StatusBadRequest, err
		}
	}
	contentType := "application/json"
	statuscode, buf, err := h.HttpDelete(uri, contentType, body)
	if err != nil {
		klog.Errorln(err)
		return statuscode, err
	}

	if len(buf) > 0 {
		err = json.Unmarshal(buf, &result)
		if err != nil {
			return statuscode, fmt.Errorf("%s", buf)
		}
	}

	return statuscode, nil
}
