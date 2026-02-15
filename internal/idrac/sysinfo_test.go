package idrac

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetSystemInfo(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/data/login" {
			http.SetCookie(w, &http.Cookie{Name: "_appwebSessionId_", Value: "sess"})
			fmt.Fprint(w, `<root><authResult>0</authResult><forwardUrl>index.html</forwardUrl></root>`)
			return
		}
		fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?>
			<root>
				<hostName>R710-TEST</hostName>
				<sysDesc>PowerEdge R710</sysDesc>
				<sysRev>II</sysRev>
				<biosVer>6.6.0</biosVer>
				<fwVersion>2.92</fwVersion>
				<LCCfwVersion>1.5.1</LCCfwVersion>
				<osName>Ubuntu 22.04</osName>
				<svcTag>ABC1234</svcTag>
			</root>`)
	}))
	defer server.Close()

	c := NewClient("localhost", "root", "calvin")
	c.baseURL = server.URL
	c.http = server.Client()
	_ = c.Login()

	info, err := c.GetSystemInfo()
	if err != nil {
		t.Fatalf("GetSystemInfo() error = %v", err)
	}

	if info.Hostname != "R710-TEST" {
		t.Errorf("Hostname = %q, want R710-TEST", info.Hostname)
	}
	if info.Model != "PowerEdge R710" {
		t.Errorf("Model = %q, want PowerEdge R710", info.Model)
	}
	if info.ServiceTag != "ABC1234" {
		t.Errorf("ServiceTag = %q, want ABC1234", info.ServiceTag)
	}
	if info.BIOSVersion != "6.6.0" {
		t.Errorf("BIOSVersion = %q, want 6.6.0", info.BIOSVersion)
	}
	if info.FWVersion != "2.92" {
		t.Errorf("FWVersion = %q, want 2.92", info.FWVersion)
	}
	if info.LCCVersion != "1.5.1" {
		t.Errorf("LCCVersion = %q, want 1.5.1", info.LCCVersion)
	}
	if info.OSName != "Ubuntu 22.04" {
		t.Errorf("OSName = %q, want Ubuntu 22.04", info.OSName)
	}
}
