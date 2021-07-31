package platform

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"cloud.google.com/go/iam"
	"cloud.google.com/go/storage"
	"github.com/gabriel-vasile/mimetype"
	"github.com/hashicorp/waypoint-plugin-sdk/terminal"
	iampb "google.golang.org/genproto/googleapis/iam/v1"
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
		Role:    role,
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

// checks to see if correct IAM roles are already set up
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

func detectMimeType(fname string, buffer []byte) string {
	if strings.HasSuffix(fname, ".css") {
		return "text/css"
	} else if strings.HasSuffix(fname, ".js") {
		return "application/javascript"
	} else if strings.HasSuffix(fname, ".map") {
		return "binary/octet-stream"
	}

	return mimetype.Detect(buffer).String()
}

func uploadFiles(
	c context.Context,
	client *storage.Client,
	bucketName string,
	buildDir string,
	subPath string,
	errors *[]string,
) []string {
	files, err := os.ReadDir(path.Join(buildDir, subPath))
	if err != nil {
		*errors = append(*errors, err.Error())
	}

	for _, file := range files {
		if file.IsDir() {
			uploadFiles(c, client, bucketName, buildDir, subPath+file.Name()+"/", errors)
			continue
		}

		f, err := os.Open(path.Join(buildDir, subPath, file.Name()))
		if err != nil {
			*errors = append(*errors, err.Error())
			continue
		}
		defer f.Close()

		wc := client.Bucket(bucketName).Object(subPath + file.Name()).NewWriter(c)
		if _, err = io.Copy(wc, f); err != nil {
			*errors = append(*errors, err.Error())
			continue
		}

		if err := wc.Close(); err != nil {
			*errors = append(*errors, err.Error())
			continue
		}

		fileInfo, _ := f.Stat()
		size := fileInfo.Size()
		buffer := make([]byte, size)

		objectMetadata := storage.ObjectAttrsToUpdate{
			ContentType: detectMimeType(fileInfo.Name(), buffer),
		}

		if _, err := client.Bucket(bucketName).Object(subPath+file.Name()).Update(c, objectMetadata); err != nil {
			*errors = append(*errors, err.Error())
		}
	}

	return *errors
}

type DeployConfig struct {
	Bucket       string `hcl:"bucket"`
	Project      string `hcl:"project"`
	Region       string `hcl:"region,optional"`
	Directory    string `hcl:"directory,optional"`
	IndexPage    string `hcl:"index,optional"`
	NotFoundPage string `hcl:"not_found,optional"`
	BaseDir      string `hcl:"base,optional"`
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
		return fmt.Errorf("expected *DeployConfig as parameter")
	}

	// validate the config
	if c.Region == "" {
		return fmt.Errorf("region must be set to a valid GCP region")
	}

	if c.Bucket == "" {
		return fmt.Errorf("bucket is a required attribute")
	}

	tmpFiles, err := os.ReadDir("/tmp")
	if err != nil {
		return fmt.Errorf("error accessing tmp directory")
	}

	tmpDir := ""

	for _, file := range tmpFiles {
		if file.IsDir() && strings.Contains(file.Name(), "waypoint") {
			tmpDir = file.Name()
			break
		}
	}

	if tmpDir == "" {
		return fmt.Errorf("could not find tmp directory for this project")
	}

	c.BaseDir = path.Join("/tmp", tmpDir)
	c.Directory = path.Join(c.BaseDir, strings.TrimLeft(c.Directory, "./"))

	_, err = os.Stat(c.Directory)

	if err != nil {
		return fmt.Errorf("directory you specified does not exist")
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
	if err != nil && !strings.Contains(err.Error(), "You already own this bucket") {
		u.Update(fmt.Sprintf("Bucket %s not found, creating new one...", p.config.Bucket))

		if err := bkt.Create(ctx, p.config.Project,
			&storage.BucketAttrs{Location: p.config.Region, LocationType: "region"}); err != nil {
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
			NotFoundPage:   p.config.NotFoundPage,
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

	fileErrors := []string{}
	uploadFiles(ctx, client, p.config.Bucket, p.config.Directory, "", &fileErrors)

	if len(fileErrors) > 0 {
		u.Step(terminal.StatusWarn, "Some static files failed to upload")
	}

	u.Step(terminal.StatusOK, "Upload of static files complete")

	return &Deployment{
		Bucket:  p.config.Bucket,
		Region:  p.config.Region,
		Project: p.config.Project,
	}, nil
}
