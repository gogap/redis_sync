package main

import (
	"github.com/codegangsta/cli"
)

type cliAction func(context *cli.Context)

func commandPush(action cliAction) cli.Command {
	return cli.Command{
		Name:   "push",
		Usage:  "Push local config's to redis",
		Action: action,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "Config",
				Usage: "defualt will read config file of redis_conf_sync.conf",
			}, cli.BoolFlag{
				Name:  "overwrite, o",
				Usage: "Force overwrite values while value is already exist, default: false",
			}, cli.BoolFlag{
				Name:  "contine, c",
				Usage: "Continue on error",
			}, cli.BoolFlag{
				Name:  "v",
				Usage: "Show process details",
			},
		},
	}
}

func commandPull(action cliAction) cli.Command {
	return cli.Command{
		Name:   "pull",
		Usage:  "Pull config's from redis",
		Action: action,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "config",
				Usage: "defualt will read config file of redis_conf_sync.conf",
			}, cli.BoolFlag{
				Name:  "contine, c",
				Usage: "Continue on error",
			}, cli.BoolFlag{
				Name:  "overwrite, o",
				Usage: "Overwrite the value of exist key",
			}, cli.BoolFlag{
				Name:  "v",
				Usage: "Show process details",
			},
		},
	}
}

func commandInit(action cliAction) cli.Command {
	return cli.Command{
		Name:   "init",
		Usage:  "Init current dir for sync data",
		Action: action,
	}
}

func commandCommit(action cliAction) cli.Command {
	return cli.Command{
		Name:   "commit",
		Usage:  "Record changes to the repository",
		Action: action,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "message, m",
				Usage: "commit message",
			},
		},
	}
}

func commandDiff(action cliAction) cli.Command {
	return cli.Command{
		Name:   "diff",
		Usage:  "Show changes between commits, commit and working tree, etc",
		Action: action,
	}
}

func commandStatus(action cliAction) cli.Command {
	return cli.Command{
		Name:   "status",
		Usage:  "Show the working tree status",
		Action: action,
	}
}
