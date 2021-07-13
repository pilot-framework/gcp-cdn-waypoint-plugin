package release

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"

	"github.com/hashicorp/waypoint-plugin-sdk/terminal"
	"github.com/pilot-framework/gcp-cdn-waypoint-plugin/platform"
)

type ReleaseConfig struct{}

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

	return nil
}

// Implement Builder
func (rm *ReleaseManager) ReleaseFunc() interface{} {
	// return a function which will be called by Waypoint
	return rm.release
}

// A BuildFunc does not have a strict signature, you can define the parameters
// you need based on the Available parameters that the Waypoint SDK provides.
// Waypoint will automatically inject parameters as specified
// in the signature at run time.
//
// Available input parameters:
// - context.Context
// - *component.Source
// - *component.JobInfo
// - *component.DeploymentConfig
// - *datadir.Project
// - *datadir.App
// - *datadir.Component
// - hclog.Logger
// - terminal.UI
// - *component.LabelSet

// In addition to default input parameters the platform.Deployment from the Deploy step
// can also be injected.
//
// The output parameters for ReleaseFunc must be a Struct which can
// be serialzied to Protocol Buffers binary format and an error.
// This Output Value will be made available for other functions
// as an input parameter.
//
// If an error is returned, Waypoint stops the execution flow and
// returns an error to the user.
func (rm *ReleaseManager) release(ctx context.Context, ui terminal.UI, target *platform.Deployment) (*Release, error) {
	u := ui.Status()
	defer u.Close()
	u.Step("", "---Releasing to Cloud CDN---")

	return &Release{}, nil
}

func GetStaticIP(ipName string) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("gcloud", "compute", "addresses", "describe", ipName, "--format=get(address)", "--global")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return "", errors.New(string(bytes.TrimSpace(stderr.Bytes())))
	}
	return string(bytes.TrimSpace(stdout.Bytes())), nil
}
