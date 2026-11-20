package xui

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

const basicServerConfig = `{"mode":"server","listen":"0.0.0.0:8443","routes":{"vpn":[{"Name":"existing-vless","Address":"host.docker.internal:30000"}]},"authorized":{"0000000000000000000000000000000000000000000000000000000000000000":["vpn"]},"padding_enabled":true,"jitter_enabled":true}`
const deviceFingerprint = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func TestStagePerDeviceACL(t *testing.T) {
	before:=[]byte(basicServerConfig)
	out,err:=AddDeviceToACL(before,deviceFingerprint,"vpn")
	if err!=nil {t.Fatal(err)}
	if !bytes.Equal(before,[]byte(basicServerConfig)) {t.Fatal("original config modified")}
	var v struct {
		Mode string `json:"mode"`
		Authorized map[string][]string `json:"authorized"`
		PaddingEnabled bool `json:"padding_enabled"`
		JitterEnabled bool `json:"jitter_enabled"`
	}
	if err:=json.Unmarshal(out,&v);err!=nil {t.Fatal(err)}
	if len(v.Authorized)!=2 || len(v.Authorized[deviceFingerprint])!=1 || v.Authorized[deviceFingerprint][0]!="vpn" {
		t.Fatal("wrong ACL")
	}
	if !v.PaddingEnabled || !v.JitterEnabled {t.Fatal("transport flags changed")}
	if _,err:=AddDeviceToACL(out,deviceFingerprint,"vpn");err==nil {t.Fatal("accepted duplicate")}
}
func TestStageACLRejectsInvalidRequests(t *testing.T) {
	cases:=[]struct{name,conf,fp,service string}{
		{"invalid fingerprint",basicServerConfig,"broken","vpn"},
		{"uppercase fingerprint",basicServerConfig,strings.ToUpper(deviceFingerprint),"vpn"},
		{"unknown route",basicServerConfig,deviceFingerprint,"unknown"},
		{"wrong mode",strings.Replace(basicServerConfig,`"mode":"server"`,`"mode":"client"`,1),deviceFingerprint,"vpn"},
		{"invalid json","{",deviceFingerprint,"vpn"},
		{"trailing document",basicServerConfig+" {}",deviceFingerprint,"vpn"},
		{"trailing garbage",basicServerConfig+" garbage",deviceFingerprint,"vpn"},
		{"missing authorized",`{"mode":"server","routes":{"vpn":[{}]}}`,deviceFingerprint,"vpn"},
	}
	for _,tc:=range cases {
		t.Run(tc.name,func(t *testing.T){
			if _,err:=AddDeviceToACL([]byte(tc.conf),tc.fp,tc.service);err==nil {t.Fatal("accepted unsafe config/ACL")}
		})
	}
}
