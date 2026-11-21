// m6-auto-registrar serves Bridge discovery and CSR issuance behind an existing
// HTTPS reverse proxy. This is NOT a 3x-ui or VPN-subscription authorization API.
package main

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/tervon-software/Tervon-TCP-Bridge/integrations/android-ingress/xui"
)

func clientCASigner(data []byte) (crypto.Signer,error) {
	block,tail:=pem.Decode(data)
	if block==nil || len(tail)!=0{return nil,errors.New("invalid existing client CA key PEM")}
	var key any
	var err error
	switch block.Type {
	case "PRIVATE KEY": key,err=x509.ParsePKCS8PrivateKey(block.Bytes)
	case "EC PRIVATE KEY": key,err=x509.ParseECPrivateKey(block.Bytes)
	default: return nil,errors.New("unsupported existing client CA private key type")
	}
	if err!=nil{return nil,errors.New("cannot parse client CA key")}
	s,ok:=key.(*ecdsa.PrivateKey)
	if !ok{return nil,errors.New("existing client CA is not ECDSA")}
	return s,nil
}

func main() {
	var root,listen,remote,serverName,service,containerID string
	flag.StringVar(&root,"root","/etc/tervon/bridge-android","existing Android Bridge root")
	flag.StringVar(&listen,"listen","","loopback address for existing HTTPS proxy (required)")
	flag.StringVar(&remote,"remote-url","","Bridge HTTPS /v2/stream endpoint advertised to Android")
	flag.StringVar(&serverName,"server-name","","server TLS name of configured Bridge")
	flag.StringVar(&service,"service","vpn","existing configured Bridge service")
	flag.StringVar(&containerID,"container-id","","running Android ingress Docker container ID")
	flag.Parse()
	if filepath.Clean(root)!=root || !filepath.IsAbs(root) || root=="/" {
		log.Fatal("invalid existing Android Bridge root")
	}
	host,port,err:=net.SplitHostPort(listen)
	if err!=nil || net.ParseIP(host)==nil || !net.ParseIP(host).IsLoopback() || port=="" {
		log.Fatal("registrar must bind a literal loopback IP:port")
	}
	if containerID=="" || serverName=="" || remote=="" {
		log.Fatal("Android ingress container and remote URL/server name required")
	}
	clientCAPEM,err:=os.ReadFile(filepath.Join(root,"config/pki/client-ca.crt"))
	if err!=nil{log.Fatal("existing client CA unavailable")}
	privatePEM,err:=os.ReadFile(filepath.Join(root,"private/client-ca.key"))
	if err!=nil{log.Fatal("existing client CA signer unavailable")}
	signer,err:=clientCASigner(privatePEM)
	if err!=nil{log.Fatal(err)}
	activator:=&xui.HotACLActivator{
		Path:filepath.Join(root,"config/server.json"),
		Service:service,
		Confirm:xui.DockerACLConfirmer{
			ContainerID:containerID,
			HostConfigDirectory:filepath.Join(root,"config"),
		}.Confirm,
	}
	enrollment:=&xui.BridgeEnrollmentHTTP{
		RemoteURL:remote,ServerName:serverName,Service:service,
		ClientCAPEM:clientCAPEM, ClientCAKey:signer,
		Activate:func(fingerprint string) error {
			ctx,cancel:=context.WithTimeout(context.Background(),15*time.Second)
			defer cancel()
			return activator.Activate(ctx,fingerprint)
		},
	}
	handler,err:=enrollment.Handler()
	if err!=nil{log.Fatal(err)}
	mux:=http.NewServeMux()
	mux.Handle("/.well-known/tervon-bridge",handler)
	ln,err:=net.Listen("tcp",listen)
	if err!=nil{log.Fatal(fmt.Errorf("cannot open registrar loopback listener: %w",err))}
	log.Print("M6_BRIDGE_REGISTRAR=LOOPBACK_LISTENING")
	srv:=&http.Server{
		Handler:mux,ReadHeaderTimeout:5*time.Second,
		IdleTimeout:15*time.Second,MaxHeaderBytes:16<<10,
	}
	log.Fatal(srv.Serve(ln))
}
