package release

import (
	"context"
	"fmt"

	"github.com/hashicorp/waypoint-plugin-sdk/component"
	"github.com/hashicorp/waypoint-plugin-sdk/terminal"
	"github.com/pilot-framework/gcp-cdn-waypoint-plugin/gcloud"
	"github.com/pilot-framework/gcp-cdn-waypoint-plugin/platform"
)

func (r *Release) URL() string { return r.Url }

var _ component.Release = (*Release)(nil)

type ReleaseConfig struct {
	Domain string `hcl:"domain"`
}

type ReleaseManager struct {
	config ReleaseConfig
}

// Implement the Destroyer interface
func (rm *ReleaseManager) DestroyFunc() interface{} {
	return rm.Destroy
}

// If an error is returned, Waypoint stops the execution flow and
// returns an error to the user.
func (rm *ReleaseManager) Destroy(ctx context.Context, ui terminal.UI, release *Release) error {
	u := ui.Status()
	defer u.Close()
	u.Step("", "---Destroying Cloud CDN resources---")

	gc := gcloud.Init(release.Project, release.Bucket)

	// DESTROY FORWARDING RULE
	u.Update("Destroying forwarding rule...")

	if gc.ForwardRule.Exists() {
		_, err := gc.ForwardRule.Destroy()
		if err != nil {
			return fmt.Errorf("failed to destroy forwarding rule: %s", err.Error())
		}
	}

	u.Step(terminal.StatusOK, "Destroyed forwarding rule")

	// DESTROY HTTPS PROXY
	u.Update("Destroying HTTPS proxy...")

	if gc.Proxy.Exists("https") {
		_, err := gc.Proxy.Destroy("https")
		if err != nil {
			return fmt.Errorf("failed to destroy HTTPS proxy: %s", err.Error())
		}
	}

	u.Step(terminal.StatusOK, "Destroyed HTTPS proxy")

	// DESTROY SSL CERT
	u.Update("Destroying SSL Certificate...")

	if gc.SSLCert.Exists() {
		_, err := gc.SSLCert.Destroy()
		if err != nil {
			return fmt.Errorf("failed to destroy SSL Certificate: %s", err.Error())
		}
	}

	u.Step(terminal.StatusOK, "Destroyed SSL Certificate")

	// DESTROY LOAD BALANCER
	u.Update("Destroying load balancer...")

	if gc.URLMap.Exists() {
		_, err := gc.URLMap.Destroy()
		if err != nil {
			return fmt.Errorf("failed to destroy load balancer: %s", err.Error())
		}
	}

	u.Step(terminal.StatusOK, "Destroyed load balancer")

	// DESTROY BACKEND BUCKET
	u.Update("Destroying backend bucket...")

	if gc.BackendBucket.Exists() {
		_, err := gc.BackendBucket.Destroy()
		if err != nil {
			return fmt.Errorf("failed to destroy backend bucket: %s", err.Error())
		}
	}

	u.Step(terminal.StatusOK, "Destroyed backend bucket")

	// DESTROY IP ADDRESS
	u.Update("Destroying IP address...")

	if gc.IP.Exists() {
		_, err := gc.IP.Destroy()
		if err != nil {
			return fmt.Errorf("failed to destroy IP address: %s", err.Error())
		}
	}

	u.Step(terminal.StatusOK, "Destroyed IP address")

	return nil
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
		return fmt.Errorf("domain is a required attribute")
	}

	return nil
}

// Implement Builder
func (rm *ReleaseManager) ReleaseFunc() interface{} {
	// return a function which will be called by Waypoint
	return rm.Release
}

func (rm *ReleaseManager) Release(ctx context.Context, ui terminal.UI, target *platform.Deployment) (*Release, error) {
	u := ui.Status()
	defer u.Close()

	if target == nil || target.Project == "" || target.Bucket == "" {
		u.Step(terminal.StatusError, "Did not receive deployment information from Platform phase")
		return nil, fmt.Errorf("aborting release phase")
	}

	u.Step("", "---Releasing to Cloud CDN---")

	gc := gcloud.Init(target.Project, target.Bucket)

	// PROVISION IP ADDRESS
	u.Update("Configuring IP Address...")

	if gc.IP.Exists() {
		u.Step(terminal.StatusOK, "Found existing external IP Address")
	} else {
		u.Update("Reserving new external IP Address...")
		_, err := gc.IP.Reserve()
		if err != nil {
			return nil, fmt.Errorf("failed to reserve IP Address: %s", err.Error())
		}

		u.Step(terminal.StatusOK, "Reserved new external IP Address")
	}

	// PROVISION BACKEND BUCKET
	u.Update("Configuring backend bucket...")

	if gc.BackendBucket.Exists() {
		u.Step(terminal.StatusOK, "Found existing backend bucket")
	} else {
		u.Update("Creating new backend bucket...")
		_, err := gc.BackendBucket.Create()
		if err != nil {
			return nil, fmt.Errorf("failed to create backend bucket: %s", err.Error())
		}

		u.Step(terminal.StatusOK, "Created new backend bucket")
	}

	// PROVISION LOAD BALANCER
	u.Update("Configuring load balancer...")

	if gc.URLMap.Exists() {
		u.Step(terminal.StatusOK, "Found existing load balancer")
	} else {
		u.Update("Creating new load balancer...")
		_, err := gc.URLMap.Create()
		if err != nil {
			return nil, fmt.Errorf("failed to create load balancer: %s", err.Error())
		}

		u.Step(terminal.StatusOK, "Created new load balancer")
	}

	// GENERATE SSL CERTIFICATE
	u.Update("Configuring SSL Certificate...")

	if gc.SSLCert.Exists() {
		u.Step(terminal.StatusOK, "Found existing SSL Certificate")
	} else {
		u.Update("Generating Google-managed SSL Certificate...")
		_, err := gc.SSLCert.Create(rm.config.Domain)
		if err != nil {
			return nil, fmt.Errorf("failed to generate SSL Certificate: %s", err.Error())
		}

		u.Step(terminal.StatusOK, "Generated Google-managed SSL Certificate")
	}

	// PROVISION HTTPS PROXY
	u.Update("Configuring HTTPS proxy...")

	if gc.Proxy.Exists("https") {
		u.Step(terminal.StatusOK, "Found existing HTTPS proxy")
	} else {
		u.Update("Creating new HTTPS proxy...")
		_, err := gc.Proxy.Create("https")
		if err != nil {
			return nil, fmt.Errorf("failed to create HTTPS proxy: %s", err.Error())
		}

		u.Step(terminal.StatusOK, "Created new HTTPS proxy")
	}

	// CREATE FORWARDING RULE
	u.Update("Configuring forwarding rules...")

	if gc.ForwardRule.Exists() {
		u.Step(terminal.StatusOK, "Found existing forwarding rule")
	} else {
		u.Update("Creating new forwarding rule...")
		_, err := gc.ForwardRule.Create()
		if err != nil {
			return nil, fmt.Errorf("failed to create forwarding rule: %s", err.Error())
		}

		u.Step(terminal.StatusOK, "Created new forwarding rule")
	}

	u.Step("", "Please allow at least 30 minutes for SSL certificate to be fully provisioned - don't forget to set up your DNS too!")

	return &Release{
		Url:     "https://" + rm.config.Domain,
		Project: target.Project,
		Bucket:  target.Bucket,
	}, nil
}
