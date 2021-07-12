package platform

import (
	"context"
	"fmt"
	"os"
	// "time"

	"cloud.google.com/go/storage"
	"github.com/hashicorp/waypoint-plugin-sdk/terminal"
)

type DeployConfig struct {
	Bucket string `hcl:"bucket"`
	Project string `hcl:"project"`
	Region string `hcl:"region,optional"`
	Directory string `hcl:"directory,optional"`
}

type Platform struct {
	config DeployConfig
}

// Implement Configurable
func (p *Platform) Config() (interface{}, error) {
	return &p.config, nil
}

// Implement ConfigurableNotify
func (p *Platform) ConfigSet(config interface{}) error {
	c, ok := config.(*DeployConfig)
	if !ok {
		// The Waypoint SDK should ensure this never gets hit
		return fmt.Errorf("Expected *DeployConfig as parameter")
	}

	// validate the config
	if c.Region == "" {
		return fmt.Errorf("Region must be set to a valid GCP region")
	}

	if c.Bucket == "" {
		return fmt.Errorf("Bucket is a required attribute")
	}

	_, err := os.Stat(c.Directory)

	if err != nil {
		return fmt.Errorf("Directory you specified does not exist")
	}

	return nil
}

// Implement Builder
func (p *Platform) DeployFunc() interface{} {
	// return a function which will be called by Waypoint
	return p.deploy
}

// this creates a new bucket in the project
// func createBucket(projectID, bucketName string) error {
// 	ctx := context.Background()
// 	client, err := storage.NewClient(ctx)
// 	if err != nil {
// 		return fmt.Errorf("storage.NewClient: %v", err)
// 	}
// 	defer client.Close()

// 	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
// 	defer cancel()

// 	bucket := client.Bucket(bucketName)
// 	if err := bucket.Create(ctx, projectID, nil); err != nil {
// 		return fmt.Errorf("Bucket(%q).Create: %v", bucketName, err)
// 	}

// 	fmt.Fprintf(w, "Bucket %v created\n", bucketName)

// 	return nil
// }

// In addition to default input parameters the registry.Artifact from the Build step
// can also be injected.
//
// The output parameters for BuildFunc must be a Struct which can
// be serialzied to Protocol Buffers binary format and an error.
// This Output Value will be made available for other functions
// as an input parameter.
// If an error is returned, Waypoint stops the execution flow and
// returns an error to the user.
func (p *Platform) deploy(ctx context.Context, ui terminal.UI) (*Deployment, error) {
	u := ui.Status()
	defer u.Close()
	u.Step("", "---Deploying Cloud Storage Assets---")

	if p.config.Directory == "" {
		p.config.Directory = "./build"
	}

	client, err := storage.NewClient(ctx)
	if err != nil {
		u.Step(terminal.StatusError, "Error connecting to Cloud Storage API")
		return nil, err
	}
	defer client.Close()

	u.Update("Configuring Cloud Storage bucket...")
	bkt := client.Bucket(p.config.Bucket)

	attrs, err := bkt.Attrs(ctx)
	if err != nil {
		u.Step(terminal.StatusError, "Error accessing bucket attributes")
		return nil, err
	}

	u.Step(terminal.StatusOK, fmt.Sprintf("Found bucket %s created at %s", attrs.Name, attrs.Created))

	return &Deployment{}, nil
}
