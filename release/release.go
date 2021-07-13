package release

import (
	"context"
	"fmt"
	"strings"

	"github.com/pilot-framework/gcp-cdn-waypoint-plugin/gcloud"
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

	gc := gcloud.Init(target.Project, target.Bucket)

	out, err := gc.IP.List()
	if err != nil {
		return nil, fmt.Errorf("failed to get listing of external IPs: %s", err.Error())
	}

	if strings.Contains(out, target.Bucket) {
		u.Step(terminal.StatusOK, "Found existing external IP address")
	} else {
		u.Step("", "Reserving external IP address")
		_, err := gc.IP.Reserve()
		if err != nil {
			return nil, fmt.Errorf("failed to get listing of external IPs: %s", err.Error())
		}
		u.Step(terminal.StatusOK, "External IP reserved")
	}

	ipAddr, err := gc.IP.GetStatic()
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
