package cmd

import (
	"fmt"

	"github.com/CodyGuo/glog"
	"github.com/brenozd/tc-docker/global"
	"github.com/brenozd/tc-docker/internal/docker"
	"github.com/brenozd/tc-docker/internal/tc"
	"github.com/spf13/cobra"
)

var debug bool

func init() {
	rootCmd.Flags().BoolVarP(&debug, "debug", "d", false, "set logger debug")
}

var rootCmd = &cobra.Command{
	Use:   "",
	Short: "",
	Long:  "",
	PreRun: func(cmd *cobra.Command, args []string) {
		if debug {
			glog.SetLevel(glog.DEBUG)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		c := docker.NewContainer(global.Ctx, global.DockerClient)
		containers, err := c.GetRunningList()
		if err != nil {
			glog.Fatal(err)
		}
		for _, container := range containers {
			err := tc.SetTC(container)
			if err != nil {
				glog.Errorf("SetTC failed, container: %s, id: %s, error: %v", container.Name, container.ID, err)
				continue
			}
			glog.Infof("SetTC success, %s", tc.GetTcString(container))
		}

		startErr := c.EventStart(func(container docker.Container) error {
			err := tc.SetTC(&container)
			if err != nil {
				return fmt.Errorf("SetTC failed, container: %s, id: %s, error: %v", container.Name, container.ID, err)
			}
			glog.Infof("AutoDiscover SetTC success, %s", tc.GetTcString(&container))
			return nil
		})
		dieErr := c.EventDie(func(container docker.Container) error {
			glog.Infof("Container stopped, name: %s, id: %s", container.Name, container.ID)
			c.RemoveIfb(container.Name)
			return c.RemoveVeth(container.Name)
		})
		for {
			select {
			case err := <-startErr:
				glog.Errorf("EventStart error: %v", err)
			case err := <-dieErr:
				glog.Errorf("EventDie error: %v", err)
			}
		}
	},
}

func Execute() error {
	return rootCmd.Execute()
}
