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
	Proxy *Proxy
	ForwardRule *ForwardRule
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
	gc.Proxy = &Proxy{g: gc}
	gc.ForwardRule = &ForwardRule{g: gc}

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

func (ip *IP) Reserve() (string, error) {
	return ip.g.Exec([]string{
		"compute",
		"addresses",
		"create", ip.g.Bucket+"-ip",
		"--network-tier=PREMIUM",
		"--ip-version=IPV4",
		"--global",
		"--project="+ip.g.Project,
	})
}

func (ip *IP) Exists() bool {
	_, err := ip.g.Exec([]string{
		"compute",
		"addresses",
		"describe",
		ip.g.Bucket+"-ip",
		"--global",
		"--project="+ip.g.Project,
	})

	if err != nil {
		return false
	} else {
		return true
	}
}

func (ip *IP) Destroy() (string, error) {
	return ip.g.Exec([]string{
		"compute",
		"addresses",
		"delete",
		ip.g.Bucket+"-ip",
		"--global",
	})
}

type BackendBucket struct {
	g *GCloud
}

func (b *BackendBucket) Create() (string, error) {
	return b.g.Exec([]string{
		"compute",
		"backend-buckets",
		"create",
		b.g.Bucket+"-backend-bucket",
		"--gcs-bucket-name="+b.g.Bucket,
		"--enable-cdn",
		"--project="+b.g.Project,
	})
}

func (b *BackendBucket) Exists() bool {
	_, err := b.g.Exec([]string{
		"compute",
		"backend-buckets",
		"describe",
		b.g.Bucket+"-backend-bucket",
	})

	if err != nil {
		return false
	} else {
		return true
	}
}

func (b *BackendBucket) Destroy() (string, error) {
	return b.g.Exec([]string{
		"compute",
		"backend-buckets",
		"delete",
		b.g.Bucket+"-backend-bucket",
	})
}

type URLMap struct {
	g *GCloud
}

func (u *URLMap) Create() (string, error) {
	return u.g.Exec([]string{
		"compute",
		"url-maps",
		"create",
		u.g.Bucket+"-lb",
		"--default-backend-bucket="+u.g.Bucket+"-backend-bucket",
		"--project="+u.g.Project,
	})
}

func (u *URLMap) Exists() bool {
	_, err := u.g.Exec([]string{
		"compute",
		"url-maps",
		"describe",
		u.g.Bucket+"-lb",
	})

	if err != nil {
		return false
	} else {
		return true
	}
}

func (u *URLMap) Destroy() (string, error) {
	return u.g.Exec([]string{
		"compute",
		"url-maps",
		"delete",
		u.g.Bucket+"-lb",
	})
}

type Proxy struct {
	g *GCloud
}

// type is a reserved word :/
func (p *Proxy) Create(which string) (string, error) {
	return p.g.Exec([]string{
		"compute", 
		"target-"+which+"-proxies",
		"create", p.g.Bucket+"-lb-proxy",
		"--url-map="+p.g.Bucket+"-lb",
		"--ssl-certificates="+p.g.Bucket+"-cert",
		"--project="+p.g.Project,
	})
}

func (p *Proxy) Exists(which string) bool {
	_, err := p.g.Exec([]string{
		"compute",
		"target-"+which+"-proxies",
		"describe",
		p.g.Bucket+"-lb-proxy",
	})

	if err != nil {
		return false
	} else {
		return true
	}
}

func (p *Proxy) Destroy(which string) (string, error) {
	return p.g.Exec([]string{
		"compute",
		"target-"+which+"-proxies",
		"delete",
		p.g.Bucket+"-lb-proxy",
	})
}

type ForwardRule struct {
	g *GCloud
}

func (f *ForwardRule) Create() (string, error) {
	return f.g.Exec([]string{
		"compute",
		"forwarding-rules",
		"create",
		f.g.Bucket+"-lb-forwarding-rule",
		"--address="+f.g.Bucket+"-ip",
		"--global",
		"--target-https-proxy="+f.g.Bucket+"-lb-proxy",
		"--ports=443",
		"--project="+f.g.Project,
	})
}

func (f *ForwardRule) Exists() bool {
	_, err := f.g.Exec([]string{
		"compute",
		"forwarding-rules",
		"describe",
		f.g.Bucket+"-lb-forwarding-rule",
		"--global",
	})

	if err != nil {
		return false
	} else {
		return true
	}
}

func (f *ForwardRule) Destroy() (string, error) {
	return f.g.Exec([]string{
		"compute",
		"forwarding-rules",
		"delete",
		f.g.Bucket+"-lb-forwarding-rule",
		"--global",
	})
}

type SSLCert struct {
	g *GCloud
}

func (s *SSLCert) Create(domain string) (string, error) {
	return s.g.Exec([]string{
		"compute",
		"ssl-certificates",
		"create", s.g.Bucket+"-cert",
		"--domains="+domain, "--global",
		"--project="+s.g.Project,
	})
}

func (s *SSLCert) Exists() bool {
	_, err := s.g.Exec([]string{
		"compute",
		"ssl-certificates",
		"describe",
		s.g.Bucket+"-cert",
	})

	if err != nil {
		return false
	} else {
		return true
	}
}

func (s *SSLCert) Destroy() (string, error) {
	return s.g.Exec([]string{
		"compute",
		"ssl-certificates",
		"delete",
		s.g.Bucket+"-cert",
	})
}