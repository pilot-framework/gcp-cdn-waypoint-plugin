package platform

import (
	"context"
	"fmt"
	"os"
	// "time"

	"cloud.google.com/go/iam"
	"cloud.google.com/go/storage"
	iampb "google.golang.org/genproto/googleapis/iam/v1"
	"github.com/hashicorp/waypoint-plugin-sdk/terminal"
)

func setPublicIAM(
	c context.Context,
	client *storage.Client,
	bucketName string,
) (bool, error) {
	policy, err := client.Bucket(bucketName).IAM().V3().Policy(c)
	if err != nil {
		return false, err
	}

	role := "roles/storage.objectViewer"
	policy.Bindings = append(policy.Bindings, &iampb.Binding{
		Role: role,
		Members: []string{iam.AllUsers},
	})

	if err := client.Bucket(bucketName).IAM().V3().SetPolicy(c, policy); err != nil {
		return false, err
	}

	return true, nil
}

func includesAllUsers(members []string) bool {
	for _, member := range members {
		if member == iam.AllUsers {
			return true
		}
	}

	return false
}

func areObjectsPublic(
	c context.Context,
	client *storage.Client,
	bucketName string,
) (bool, error) {
	policy, err := client.Bucket(bucketName).IAM().V3().Policy(c)
	if err != nil {
		return false, err
	}

	for _, binding := range policy.Bindings {
		if binding.Role == "roles/storage.objectViewer" && includesAllUsers(binding.Members) {
			return true, nil
		}
	}

	return false, nil
}

type DeployConfig struct {
	Bucket string `hcl:"bucket"`
	Project string `hcl:"project"`
	Region string `hcl:"region,optional"`
	Directory string `hcl:"directory,optional"`
	IndexPage string `hcl:"index,optional"`
	NotFoundPage string `hcl:"not_found,optional"`
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

	if c.Directory != "" {
		_, err := os.Stat(c.Directory)

		if err != nil {
			return fmt.Errorf("Directory you specified does not exist")
		}
	}

	return nil
}

// Implement Builder
func (p *Platform) DeployFunc() interface{} {
	// return a function which will be called by Waypoint
	return p.deploy
}

// If an error is returned, Waypoint stops the execution flow and
// returns an error to the user.
func (p *Platform) deploy(ctx context.Context, ui terminal.UI) (*Deployment, error) {
	u := ui.Status()
	defer u.Close()
	u.Step("", "---Deploying Cloud Storage Assets---")

	// configure defaults
	if p.config.Directory == "" {
		p.config.Directory = "./build"
	}

	if p.config.IndexPage == "" {
		p.config.IndexPage = "index.html"
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
	// if this errors out, bucket doesn't exist
	if err != nil {
		u.Update(fmt.Sprintf("Bucket %s not found, creating new one...", p.config.Bucket))
		
		if err := bkt.Create(ctx, p.config.Project, nil); err != nil {
			u.Step(terminal.StatusError, "Error creating new bucket")
			return nil, err
		}

		newBktAttrs := storage.BucketAttrsToUpdate{
			UniformBucketLevelAccess: &storage.UniformBucketLevelAccess{
				Enabled: true,
			},
		}

		if _, err := bkt.Update(ctx, newBktAttrs); err != nil {
			u.Step(terminal.StatusError, fmt.Sprintf("Error configuring %s to be uniformly accessible", attrs.Name))
			return nil, err
		}

		u.Step(terminal.StatusOK, fmt.Sprintf("Bucket %s successfully created", p.config.Bucket))
	} else {
		u.Step(terminal.StatusOK, fmt.Sprintf("Found existing bucket %s", attrs.Name))
	}

	u.Update("Configuring bucket for website hosting...")

	bktAttrsToUpdate := storage.BucketAttrsToUpdate{
		Website: &storage.BucketWebsite{
			MainPageSuffix: p.config.IndexPage,
			NotFoundPage: p.config.NotFoundPage,
		},
	}

	if _, err := bkt.Update(ctx, bktAttrsToUpdate); err != nil {
		u.Step(terminal.StatusError, fmt.Sprintf("Error configuring %s to host static content", p.config.Bucket))
		return nil, err
	}

	// set all objects to be publicly readable
	public, err := areObjectsPublic(ctx, client, p.config.Bucket)
	if err != nil {
		u.Step(terminal.StatusError, "Error accessing bucket's IAM policy")
		return nil, err
	}

	if !public {
		if _, err := setPublicIAM(ctx, client, p.config.Bucket); err != nil {
			u.Step(terminal.StatusError, fmt.Sprintf("Error configuring %s objects to be publicly accessible", p.config.Bucket))
			return nil, err
		}
	}

	u.Step(terminal.StatusOK, fmt.Sprintf("Objects within %s are publicly accessible", p.config.Bucket))

	u.Update("Uploading static files...")

	// TODO

	return &Deployment{}, nil
}
