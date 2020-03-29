package client

import "github.com/urfave/cli"

var clientCMD = cli.Command{
	Name:  "client",
	Usage: "BitXHub client command",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "gateway",
			Usage: "Specific gateway address",
			Value: "localhost:9091",
		},
		cli.StringFlag{
			Name:  "grpc",
			Usage: "Specific grpc address",
			Value: "localhost:60011",
		},
	},
	Subcommands: cli.Commands{
		accountCMD(),
		appchainCMD(),
		chainCMD(),
		blockCMD(),
		networkCMD(),
		receiptCMD(),
		ruleCMD(),
		txCMD(),
		interchainCMD(),
	},
}

func LoadClientCMD() cli.Command {
	return clientCMD
}
