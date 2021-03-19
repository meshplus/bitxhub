package client

import "github.com/urfave/cli"

var clientCMD = cli.Command{
	Name:  "client",
	Usage: "BitXHub client command",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "gateway",
			Usage: "Specific gateway address",
			Value: "http://localhost:9091/v1/",
		},
		cli.StringFlag{
			Name:  "cert",
			Usage: "Specific ca cert file if https is enabled",
		},
	},
	Subcommands: cli.Commands{
		accountCMD(),
		chainCMD(),
		blockCMD(),
		networkCMD(),
		receiptCMD(),
		txCMD(),
		validatorsCMD(),
		delVPNodeCMD(),
		governanceCMD(),
	},
}

func LoadClientCMD() cli.Command {
	return clientCMD
}
