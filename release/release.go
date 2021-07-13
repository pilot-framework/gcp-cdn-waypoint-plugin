package release

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/hashicorp/waypoint-plugin-sdk/terminal"
	"github.com/pilot-framework/gcp-cdn-waypoint-plugin/platform"
)

type GCloud struct {
	Project string
	Bucket string
}

func (g *GCloud) Exec(args []string) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("gcloud", args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return "", errors.New(string(bytes.TrimSpace(stderr.Bytes())))
	}
	return string(bytes.TrimSpace(stdout.Bytes())), nil
}

func (g *GCloud) ListIPAddresses() (string, error) {
	return g.Exec([]string{
		"compute", "addresses", "list", "--project="+g.Project,
	})
}

func (g *GCloud) GetStaticIP() (string, error) {
	return g.Exec([]string{
		"compute", "addresses", "describe", g.Bucket+"-ip", "--format=get(address)", "--global", "--project="+g.Project,
	})
}

func (g * GCloud) ReserveIPAddress() (string, error) {
	return g.Exec([]string{
		"compute", "addresses", "create", g.Bucket+"-ip", "--network-tier=PREMIUM", "--ip-version=IPV4", "--global", "--project="+g.Project,
	})
} 

func (g *GCloud) CreateBackendBucket() (string, error) {
	return g.Exec([]string{
		"compute", "backend-buckets", g.Bucket+"-backend-bucket", "--gcs-bucket-name="+"b", "--enable-cdn", "--project="+g.Project,
	})
}

func (g *GCloud) CreateURLMap() (string, error) {
	return g.Exec([]string{
		"compute", "url-maps", "create", g.Bucket+"-lb", "--default-backend-bucket="+g.Bucket+"-backend-bucket", "--project="+g.Project,
	})
}

func (g *GCloud) CreateSSLCert(domain string) (string, error) {
	return g.Exec([]string{
		"compute", "ssl-certificates", "create", g.Bucket+"-cert", "--domains="+domain, "--global", "--project="+g.Project,
	})
}

type ReleaseConfig struct {
	Domain string `hcl:"domain"`
}

type ReleaseManager struct {
	config ReleaseConfig
}

// Implement Configurable
func (rm *ReleaseManager) Config() (interface{}, error) {
	return &rm.config, nil
}

// Implement ConfigurableNotify
func (rm *ReleaseManager) ConfigSet(config interface{}) error {
	_, ok := config.(*ReleaseConfig)
	if !ok {
		// The Waypoint SDK should ensure this never gets hit
		return fmt.Errorf("Expected *ReleaseConfig as parameter")
	}

	// validate the config
	if rm.config.Domain == "" {
		return fmt.Errorf("Domain is a required attribute")
	}

	return nil
}

// Implement Builder
func (rm *ReleaseManager) ReleaseFunc() interface{} {
	// return a function which will be called by Waypoint
	return rm.release
}

func (rm *ReleaseManager) release(ctx context.Context, ui terminal.UI, target *platform.Deployment) (*Release, error) {
	u := ui.Status()
	defer u.Close()
	u.Step("", "---Releasing to Cloud CDN---")

	gc := &GCloud{Bucket: target.Bucket, Project: target.Project}

	out, err := gc.ListIPAddresses()
	if err != nil {
		return nil, fmt.Errorf("failed to get listing of external IPs: %s", err.Error())
	}

	if strings.Contains(out, target.Bucket) {
		u.Step(terminal.StatusOK, "Found existing external IP address")
	} else {
		u.Step("", "Reserving external IP address")
		_, err := gc.ReserveIPAddress()
		if err != nil {
			return nil, fmt.Errorf("failed to get listing of external IPs: %s", err.Error())
		}
		u.Step(terminal.StatusOK, "External IP reserved")
	}

	ipAddr, err := gc.GetStaticIP()
	if err != nil {
		return nil, fmt.Errorf("failed to get external IP: %s", err.Error())
	}

	//TODO: load balance configuration
	u.Step("", "ipaddress="+ipAddr)

	return &Release{}, nil
}

//TODO: configure target https proxy
// see - gcloud compute target-https-proxies create --help
//TODO: configure forwarding rule to reserved ip and target proxy
