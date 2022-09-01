package traefik_cf_real_ip

import (
	"context"
    "fmt"
    "net/netip"
	"net"
	"net/http"
)

// Config the plugin configuration.
type Config struct {
	CloudFlareIPs []string `yaml:"cloudFlareIps"`
	CloudFlareHeader string `yaml:"cloudFlareHeader"`
	DestinationHeader string `yaml:"destHeader"`
	PrependIP bool `yaml:"prependIp"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		// https://www.cloudflare.com/ips-v4
		CloudFlareIPs: []string{"173.245.48.0/20","103.21.244.0/22","103.22.200.0/22","103.31.4.0/22","141.101.64.0/18","108.162.192.0/18","190.93.240.0/20","188.114.96.0/20","197.234.240.0/22","198.41.128.0/17","162.158.0.0/15","104.16.0.0/13","104.24.0.0/14","172.64.0.0/13","131.0.72.0/22"},
		CloudFlareHeader: "CF-Connecting-IP",
		DestinationHeader: "X-Forwarded-For",
		PrependIP: false,
	}
}

// GetRealIP Define plugin
type GetRealIP struct {
	next  http.Handler
	name  string
	parsedPrefxies     []netip.Prefix
	cfHeader string
	destinationHeader string
	prependIp bool
}    


// New creates and returns a new realip plugin instance.
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	parsedPrefixes := []netip.Prefix{}

	for _, element := range config.CloudFlareIPs {
		network, err := netip.ParsePrefix(element)
		if err != nil {
			panic(err)
		}
		parsedPrefixes = append(parsedPrefixes, network)
	}

	fmt.Printf("CF RealIP Plugin Config: Config：'%v'\n", config)

	return &GetRealIP{
		next:  next,
		name:  name,
		parsedPrefxies: parsedPrefixes,
		cfHeader: config.CloudFlareHeader,
		destinationHeader: config.DestinationHeader,
		prependIp: config.PrependIP,
	}, nil
}

func (g *GetRealIP) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	fmt.Println("remoteaddr", req.RemoteAddr)
	fmt.Println("Header: ", g.cfHeader)
	
	ip, err := netip.ParseAddr(req.RemoteAddr)

    if err != nil {
        fmt.Println(err)
		g.next.ServeHTTP(rw, req)
		return
    }

	for _, element := range g.parsedPrefxies {
		if element.Contains(ip) {
			val := req.Header.Get(g.cfHeader)

			if val == "" {
				fmt.Println("Cloudlfare header not present")
				g.next.ServeHTTP(rw, req)
				return
			}

			if g.prependIp {
				req.Header.Set(g.destinationHeader, val + "," + req.Header.Get(g.destinationHeader))
			} else {
				req.Header.Set(g.destinationHeader, val)
			}
			g.next.ServeHTTP(rw, req)
			return
		}
	}
	g.next.ServeHTTP(rw, req)
	return
}
