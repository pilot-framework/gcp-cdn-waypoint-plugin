package gcloud

import (
	"bytes"
	"errors"
	"os/exec"
)

type GCloud struct {
	Project string
	Bucket string
	IP *IP
	BackendBucket *BackendBucket
	URLMap *URLMap
	SSLCert *SSLCert
}

func Init(project, bucket string) *GCloud {
	gc := &GCloud{
		Project: project,
		Bucket: bucket,
	}

	gc.IP = &IP{g: gc}
	gc.BackendBucket = &BackendBucket{g: gc}
	gc.URLMap = &URLMap{g: gc}
	gc.SSLCert = &SSLCert{g: gc}

	return gc
}

func (g *GCloud) Exec(args []string) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("gcloud", args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return "", errors.New(string(bytes.TrimSpace(stderr.Bytes())))
	}
	return string(bytes.TrimSpace(stdout.Bytes())), nil
}

type IP struct {
	g *GCloud
}

func (ip *IP) List() (string, error) {
	return ip.g.Exec([]string{
		"compute", "addresses", "list", "--project="+ip.g.Project,
	})
}

func (ip *IP) GetStatic() (string, error) {
	return ip.g.Exec([]string{
		"compute", "addresses", "describe", ip.g.Bucket+"-ip", "--format=get(address)", "--global", "--project="+ip.g.Project,
	})
}

func (ip *IP) Reserve() (string, error) {
	return ip.g.Exec([]string{
		"compute", "addresses", "create", ip.g.Bucket+"-ip", "--network-tier=PREMIUM", "--ip-version=IPV4", "--global", "--project="+ip.g.Project,
	})
}

type BackendBucket struct {
	g *GCloud
}

func (b *BackendBucket) Create() (string, error) {
	return b.g.Exec([]string{
		"compute", "backend-buckets", b.g.Bucket+"-backend-bucket", "--gcs-bucket-name="+b.g.Bucket, "--enable-cdn", "--project="+b.g.Project,
	})
}

func (b *BackendBucket) Exists() bool {
	_, err := b.g.Exec([]string{
		"compute", "backend-buckets", "describe", b.g.Bucket+"-backend-bucket",
	})

	if err != nil {
		return false
	} else {
		return true
	}
}

type URLMap struct {
	g *GCloud
}

func (u *URLMap) Create() (string, error) {
	return u.g.Exec([]string{
		"compute", "url-maps", "create", u.g.Bucket+"-lb", "--default-backend-bucket="+u.g.Bucket+"-backend-bucket", "--project="+u.g.Project,
	})
}

func (u *URLMap) Exists() bool {
	_, err := u.g.Exec([]string{
		"compute", "url-maps", "describe", u.g.Bucket+"-lb",
	})

	if err != nil {
		return false
	} else {
		return true
	}
}

type SSLCert struct {
	g *GCloud
}

func (s *SSLCert) Create(domain string) (string, error) {
	return s.g.Exec([]string{
		"compute", "ssl-certificates", "create", s.g.Bucket+"-cert", "--domains="+domain, "--global", "--project="+s.g.Project,
	})
}

func (s *SSLCert) Exists() bool {
	_, err := s.g.Exec([]string{
		"compute", "ssl-certificates", "describe", s.g.Bucket+"-cert",
	})

	if err != nil {
		return false
	} else {
		return true
	}
}