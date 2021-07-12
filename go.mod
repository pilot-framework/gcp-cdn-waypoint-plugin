module github.com/pilot-framework/gcp-cdn-waypoint-plugin

go 1.14

require (
	github.com/golang/protobuf v1.5.2
	github.com/hashicorp/waypoint-plugin-sdk v0.0.0-20210625180209-eda7ae600c2d
	github.com/mitchellh/go-glint v0.0.0-20201015034436-f80573c636de
	google.golang.org/protobuf v1.26.0
)

// replace github.com/hashicorp/waypoint-plugin-sdk => ../../waypoint-plugin-sdk
