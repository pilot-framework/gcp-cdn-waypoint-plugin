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
		return fmt.Errorf("expected *ReleaseConfig as parameter")
	}

	// validate the config
	if rm.config.Domain == "" {
		return fmt.Errorf("eomain is a required attribute")
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

	out, err := ListIPAddresses(target.Project)
	if err != nil {
		return nil, fmt.Errorf("failed to get listing of external IPs: %s", err.Error())
	}

	if strings.Contains(out, target.Bucket) {
		u.Step(terminal.StatusOK, "Found existing external IP address")
	} else {
		u.Step("", "Reserving external IP address")
		_, err := ReserveIPAddress(target.Bucket, target.Project)
		if err != nil {
			return nil, fmt.Errorf("failed to reserve IP address: %s", err.Error())
		}
		u.Step(terminal.StatusOK, "External IP reserved")
	}

	// Uncomment if the static IP address is needed during execution
	// ipAddr, err := GetStaticIP(target.Bucket, target.Project)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to get external IP: %s", err.Error())
	// }

	u.Step("", "Creating Backend Bucket")
	_, err = CreateBackendBucket(target.Bucket, target.Project)
	if err != nil {
		return nil, fmt.Errorf("failed to create backed bucket: %s", err.Error())
	}
	u.Step(terminal.StatusOK, "Backend Bucket Created")

	u.Step("", "Creating Load Balancer")
	_, err = CreateURLMap(target.Bucket, target.Project)
	if err != nil {
		return nil, fmt.Errorf("failed to create load balancer: %s", err.Error())
	}
	u.Step(terminal.StatusOK, "Load Balancer Created")

	u.Step("", "Generating managed SSL Certificate")
	_, err = CreateSSLCert(target.Bucket, target.Project, rm.config.Domain)
	if err != nil {
		return nil, fmt.Errorf("failed to generate SSL certificate: %s", err.Error())
	}
	u.Step(terminal.StatusOK, "SSL Certificate Generated")

	u.Step("", "Creating HTTPS Target Proxy")
	_, err = CreateHTTPSProxy(target.Bucket, target.Project)
	if err != nil {
		return nil, fmt.Errorf("failed to create target proxy: %s", err.Error())
	}
	u.Step(terminal.StatusOK, "HTTPS Target Proxy Created")

	u.Step("", "Creating Forwarding Rule")
	_, err = CreateForwardingRule(target.Bucket, target.Project)
	if err != nil {
		return nil, fmt.Errorf("failed to create forwarding rule: %s", err.Error())
	}
	u.Step(terminal.StatusOK, "Forwarding Rule Created")

	return &Release{}, nil
}

func GetStaticIP(b, projID string) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("gcloud", "compute", "addresses", "describe", b+"-ip", "--format=get(address)", "--global", "--project="+projID)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return "", errors.New(string(bytes.TrimSpace(stderr.Bytes())))
	}
	return string(bytes.TrimSpace(stdout.Bytes())), nil
}

func ListIPAddresses(projID string) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("gcloud", "compute", "addresses", "list", "--project="+projID)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return "", errors.New(string(bytes.TrimSpace(stderr.Bytes())))
	}
	return string(bytes.TrimSpace(stdout.Bytes())), nil
}

func ReserveIPAddress(b, projID string) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("gcloud", "compute", "addresses", "create", b+"-ip", "--network-tier=PREMIUM", "--ip-version=IPV4", "--global", "--project="+projID)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return "", errors.New(string(bytes.TrimSpace(stderr.Bytes())))
	}
	return string(bytes.TrimSpace(stdout.Bytes())), nil
}

func CreateBackendBucket(b, projID string) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("gcloud", "compute", "backend-buckets", "create", b+"-backend-bucket", "--gcs-bucket-name="+b, "--enable-cdn", "--project="+projID)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return "", errors.New(string(bytes.TrimSpace(stderr.Bytes())))
	}
	return string(bytes.TrimSpace(stdout.Bytes())), nil
}

func CreateURLMap(b, projID string) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("gcloud", "compute", "url-maps", "create", b+"-lb", "--default-backend-bucket="+b+"-backend-bucket", "--project="+projID)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return "", errors.New(string(bytes.TrimSpace(stderr.Bytes())))
	}
	return string(bytes.TrimSpace(stdout.Bytes())), nil
}

func CreateSSLCert(b, projID, domain string) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("gcloud", "compute", "ssl-certificates", "create", b+"-cert", "--domains="+domain, "--global", "--project="+projID)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return "", errors.New(string(bytes.TrimSpace(stderr.Bytes())))
	}
	return string(bytes.TrimSpace(stdout.Bytes())), nil
}

func CreateHTTPSProxy(b, projID string) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("gcloud", "compute", "target-https-proxies", "create", b+"-lb-proxy", "--url-map="+b+"-lb", "--ssl-certificates="+b+"-cert", "--project="+projID)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return "", errors.New(string(bytes.TrimSpace(stderr.Bytes())))
	}
	return string(bytes.TrimSpace(stdout.Bytes())), nil
}

func CreateForwardingRule(b, projID string) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("gcloud", "compute", "forwarding-rules", "create", b+"-lb-forwarding-rule", "--address="+b+"-ip", "--global", "--target-https-proxy="+b+"-lb-proxy", "--ports=443", "--project="+projID)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return "", errors.New(string(bytes.TrimSpace(stderr.Bytes())))
	}
	return string(bytes.TrimSpace(stdout.Bytes())), nil
}
