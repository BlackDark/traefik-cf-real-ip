package traefik_cf_real_ip_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	plugin "github.com/BlackDark/traefik-cf-real-ip"
)

func TestDefaultConfig(t *testing.T) {
	cfg := plugin.CreateConfig()
	cfg.CloudFlareIPs = []string{"1.1.1.1/24"}

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {})

	handler, err := plugin.New(ctx, next, cfg, "traefik-get-real-ip")
	if err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		xff          string
		realIPHeader string 
		realIP       string 
		desc         string
		expected     string
		remoteAddr   string
	}{
		{
			xff:          "10.0.0.2",
			realIPHeader: "CF-Connecting-IP",
			realIP:       "10.1.1.1",
			remoteAddr:   "1.1.1.1",
			expected:     "10.1.1.1",
			desc: "Should correctly set from CF Header",
		},
		{
			xff:          "10.0.0.2",
			realIPHeader: "Cf-Connecting-Ip-WRONG",
			realIP:       "80.1.1.1",
			remoteAddr:   "1.1.1.1",
			expected:     "10.0.0.2",
			desc: "CF Header falsy - keep XFF header",
		},
		{
			xff:          "10.0.0.2",
			realIPHeader: "Cf-Connecting-Ip",
			realIP:       "80.1.1.1",
			remoteAddr:   "10.1.1.1",
			expected:     "10.0.0.2",
			desc: "Not allowed range - Keep XFF header",
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			reorder := httptest.NewRecorder()

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost", nil)
			if err != nil {
				t.Fatal(err)
			}
			req.RemoteAddr = test.remoteAddr
			req.Header.Set(test.realIPHeader, test.realIP)
			req.Header.Set("X-Forwarded-For", test.xff)

			handler.ServeHTTP(reorder, req)

			assertHeader(t, req, "X-Forwarded-For", test.expected)
		})
	}
}

func TestPrependConfig(t *testing.T) {
	cfg := plugin.CreateConfig()
	cfg.PrependIP = true
	cfg.CloudFlareIPs = []string{"1.1.1.1/24"}

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {})

	handler, err := plugin.New(ctx, next, cfg, "traefik-get-real-ip")
	if err != nil {
		t.Fatal(err)
	}
	reorder := httptest.NewRecorder()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost", nil)
	if err != nil {
		t.Fatal(err)
	}

	req.RemoteAddr = "1.1.1.1"
	req.Header.Set("CF-Connecting-IP", "10.1.1.1")
	req.Header.Set("X-Forwarded-For", "10.0.0.2")

	handler.ServeHTTP(reorder, req)

	assertHeader(t, req, "X-Forwarded-For", "10.1.1.1,10.0.0.2")
}

func assertHeader(t *testing.T, req *http.Request, key, expected string) {
	t.Helper()
	if req.Header.Get(key) != expected {
		t.Errorf("invalid header value: %s", req.Header.Get(key))
	}
}
