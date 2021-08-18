# GCP Cloud CDN Waypoint Plugin

This is a Deploy/Release Waypoint Plugin that provisions the necessary resources to deploy static files on GCP Cloud Storage and Cloud CDN. Requires a build directory of static files within a project. Should be used with Pilot or a remote Waypoint configuration.

## Steps

You can run the Makefile to compile the plugin, the `Makefile` will build the plugin for all architectures.

```shell
cd ../destination_folder

make
```

```shell
Build Protos
protoc -I . --go_out=plugins=grpc:. --go_opt=paths=source_relative ./platform/output.proto
protoc -I . --go_out=plugins=grpc:. --go_opt=paths=source_relative ./release/output.proto

Compile Plugin
# Clear the output
rm -rf ./bin
GOOS=linux GOARCH=amd64 go build -o ./bin/linux_amd64/waypoint-plugin-pilot-cloud-cdn ./main.go 
GOOS=darwin GOARCH=amd64 go build -o ./bin/darwin_amd64/waypoint-plugin-pilot-cloud-cdn ./main.go 
GOOS=windows GOARCH=amd64 go build -o ./bin/windows_amd64/waypoint-plugin-pilot-cloud-cdn.exe ./main.go 
GOOS=windows GOARCH=386 go build -o ./bin/windows_386/waypoint-plugin-pilot-cloud-cdn.exe ./main.go 
```

## Building with Docker

To build plugins for release you can use the `build-docker` Makefile target, this will 
build your plugin for all architectures and create zipped artifacts which can be uploaded
to an artifact manager such as GitHub releases.

The built artifacts will be output in the `./releases` folder.

```shell
make build-docker

rm -rf ./releases
DOCKER_BUILDKIT=1 docker build --output releases --progress=plain .
#1 [internal] load .dockerignore
#1 transferring context: 2B done
#1 DONE 0.0s

#...

#14 [export_stage 1/1] COPY --from=build /go/plugin/bin/*.zip .
#14 DONE 0.1s

#15 exporting to client
#15 copying files 36.45MB 0.1s done
#15 DONE 0.1s
```
