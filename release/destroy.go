package release

import (
	"context"
	"fmt"

	"github.com/hashicorp/waypoint-plugin-sdk/terminal"
	"github.com/pilot-framework/gcp-cdn-waypoint-plugin/gcloud"
)

// Implement the Destroyer interface
func (rm *ReleaseManager) DestroyFunc() interface{} {
	return rm.destroy
}

// If an error is returned, Waypoint stops the execution flow and
// returns an error to the user.
func (rm *ReleaseManager) destroy(ctx context.Context, ui terminal.UI, release *Release) error {
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
