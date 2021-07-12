module github.com/pilot-framework/gcp-cdn-waypoint-plugin

go 1.14

require (
	cloud.google.com/go v0.86.0 // indirect
	cloud.google.com/go/storage v1.10.0
	github.com/hashicorp/waypoint-plugin-sdk v0.0.0-20210625180209-eda7ae600c2d
	golang.org/x/sys v0.0.0-20210630005230-0f9fa26af87c // indirect
	google.golang.org/protobuf v1.27.1
)

// replace github.com/hashicorp/waypoint-plugin-sdk => ../../waypoint-plugin-sdk
