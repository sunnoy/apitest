package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"github.com/gin-gonic/gin"
	"github.com/go-openapi/spec"
	"github.com/golang/glog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"net/http"
	"os"
)

func main() {
	//认证API server作为client的证书
	caCrt := GetApiServerCa()

	// 手动调用 kubectl get --raw "/apis/api.ytool.io/v1"
	routers := InitRouter()
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM([]byte(caCrt))
	s := &http.Server{
		Addr:    ":443",
		Handler: routers,
		TLSConfig: &tls.Config{
			ClientCAs:  pool,
			ClientAuth: tls.RequireAndVerifyClientCert,
		},
	}
	// https 监听
	/*
		#安装openssl
		yum install openssl
		#创建根证书密钥文件
		openssl genrsa -out ca.key
		#创建根证书的申请文件
		openssl req -new -key ca.key -out ca.csr
		#创建一个自当前日期起为期十年的根证书root.crt
		openssl x509 -req -days 3650 -sha1 -extensions v3_ca -signkey ca.key -in ca.csr -out ca.crt
	*/
	err := s.ListenAndServeTLS("ca.crt", "ca.key")
	if err != nil {
		glog.Fatalf("server error %v \r\n", err)
	}
}

func GetApiServerCa() string {
	cm, err := GetK8sClient().CoreV1().ConfigMaps("kube-system").Get(context.TODO(), "extension-apiserver-authentication", metav1.GetOptions{})
	if err != nil {
		glog.Fatal(err)
	}
	ca := cm.Data["requestheader-client-ca-file"]
	if ca == "" {
		glog.Fatal("ca is empty")
	}
	return ca
}

func InitRouter() *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.GET("openapi/v2", func(context *gin.Context) {
		context.JSON(http.StatusOK, &spec.Swagger{})
	})

	group := r.Group("apis/api.ytool.io/v1")
	group.GET("/hello", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	return r
}

func GetK8sClient() *kubernetes.Clientset {
	var config *rest.Config
	var err error
	kubeconfig, ok := os.LookupEnv("kubeconfig")
	if ok {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		config, err = rest.InClusterConfig()
	}
	if err != nil {
		glog.Fatal(err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		glog.Fatal(err)
	}
	return clientset
}
