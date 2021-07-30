module github.com/pilot-framework/gcp-cdn-waypoint-plugin

go 1.14

require (
	cloud.google.com/go v0.86.0
	cloud.google.com/go/storage v1.10.0
	github.com/gabriel-vasile/mimetype v1.3.1 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/hashicorp/waypoint-plugin-sdk v0.0.0-20210625180209-eda7ae600c2d
	golang.org/x/net v0.0.0-20210726213435-c6fcb2dbf985 // indirect
	golang.org/x/sys v0.0.0-20210630005230-0f9fa26af87c // indirect
	google.golang.org/api v0.50.0
	google.golang.org/genproto v0.0.0-20210701133433-6b8dcf568a95
	google.golang.org/protobuf v1.27.1
)

// replace github.com/hashicorp/waypoint-plugin-sdk => ../../waypoint-plugin-sdk
