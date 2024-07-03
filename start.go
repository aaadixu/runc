package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/opencontainers/runc/libcontainer"
	"github.com/urfave/cli"
)

var startCommand = cli.Command{
	Name:  "start",
	Usage: "executes the user defined process in a created container",
	ArgsUsage: `<container-id>

Where "<container-id>" is your name for the instance of the container that you
are starting. The name you provide for the container instance must be unique on
your host.`,
	Description: `The start command executes the user defined process in a created container.`,
	Action: func(context *cli.Context) error {
		if err := checkArgs(context, 1, exactArgs); err != nil {
			return err
		}
		// 获取container
		container, err := getContainer(context)
		if err != nil {
			return err
		}
		status, err := container.Status()
		if err != nil {
			return err
		}
		switch status {
		case libcontainer.Created:
			// start 前对应的状态就是容器环境已经创建好了，在等待start, 然后切换至running
			notifySocket, err := notifySocketStart(context, os.Getenv("NOTIFY_SOCKET"), container.ID())
			if err != nil {
				return err
			}
			// 执行exec 容器进程替代init
			if err := container.Exec(); err != nil {
				return err
			}
			if notifySocket != nil {
				return notifySocket.waitForContainer(container)
			}
			return nil
		case libcontainer.Stopped:
			return errors.New("cannot start a container that has stopped")
		case libcontainer.Running:
			return errors.New("cannot start an already running container")
		default:
			return fmt.Errorf("cannot start a container in the %s state", status)
		}
	},
}
