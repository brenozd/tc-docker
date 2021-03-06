package main

import (
	"context"

	"github.com/CodyGuo/glog"
	"github.com/brenozd/tc-docker/cmd"
	"github.com/brenozd/tc-docker/global"
	"github.com/docker/docker/client"
)

func init() {
	err := setupSetting()
	if err != nil {
		glog.Fatalf("init.setupSetting error: %v", err)
	}
}

func main() {
	// glog.SetLevel(glog.DEBUG)
	if err := cmd.Execute(); err != nil {
		glog.Fatal(err)
	}
}

func setupSetting() error {
	var err error
	global.DockerClient, err = client.NewEnvClient()
	if err != nil {
		return err
	}
	global.Ctx = context.Background()
	return nil
}
