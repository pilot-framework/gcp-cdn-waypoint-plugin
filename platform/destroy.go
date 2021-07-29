package platform

import (
	"context"
	"fmt"

	"cloud.google.com/go/storage"
	"github.com/hashicorp/waypoint-plugin-sdk/terminal"
	"google.golang.org/api/iterator"
)

// Implement the Destroyer interface
func (p *Platform) DestroyFunc() interface{} {
	return p.destroy
}

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

	// If a bucket already doesn't exist, just short circuit
	_, err = client.Bucket(p.config.Bucket).Attrs(ctx)
	if err == storage.ErrBucketNotExist {
		u.Step(terminal.StatusOK, "Successfully destroyed Cloud Storage Assets")
		return nil
	}

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
