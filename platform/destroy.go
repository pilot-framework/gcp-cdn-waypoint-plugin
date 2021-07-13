package platform

import (
	"context"
	"fmt"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
	"github.com/hashicorp/waypoint-plugin-sdk/terminal"
)

// Implement the Destroyer interface
func (p *Platform) DestroyFunc() interface{} {
	return p.destroy
}

// A DestroyFunc does not have a strict signature, you can define the parameters
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
//
// In addition to default input parameters the Deployment from the DeployFunc step
// can also be injected.
//
// The output parameters for PushFunc must be a Struct which can
// be serialzied to Protocol Buffers binary format and an error.
// This Output Value will be made available for other functions
// as an input parameter.
//
// If an error is returned, Waypoint stops the execution flow and
// returns an error to the user.
func (p *Platform) destroy(ctx context.Context, ui terminal.UI) error {
	u := ui.Status()
	defer u.Close()
	u.Step("", "---Destroying Cloud Storage Assets---")

	client, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to Google Cloud: %s", err.Error())
	}
	defer client.Close()

	u.Update("Destroying objects...")

	it := client.Bucket(p.config.Bucket).Objects(ctx, nil)
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to list object: %s", err.Error())
		}

		obj := client.Bucket(p.config.Bucket).Object(attrs.Name)
		if err := obj.Delete(ctx); err != nil {
			return fmt.Errorf("failed to destroy object: %s", err.Error())
		}
	}

	u.Step("", "Destroyed objects")

	u.Update("Destroying bucket...")

	bkt := client.Bucket(p.config.Bucket)
	if err := bkt.Delete(ctx); err != nil {
		return fmt.Errorf("failed to destroy bucket: %s", err.Error())
	}

	u.Step(terminal.StatusOK, "Successfully destroyed Cloud Storage Assets")

	return nil
}
