package main

import (
	"fmt"
	"math/big"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/meshplus/bitxhub"
	"github.com/meshplus/bitxhub-core/agency"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub/api/gateway"
	"github.com/meshplus/bitxhub/api/grpc"
	"github.com/meshplus/bitxhub/api/jsonrpc"
	"github.com/meshplus/bitxhub/internal/app"
	"github.com/meshplus/bitxhub/internal/coreapi"
	"github.com/meshplus/bitxhub/internal/loggers"
	"github.com/meshplus/bitxhub/internal/profile"
	"github.com/meshplus/bitxhub/internal/repo"
	types2 "github.com/meshplus/eth-kit/types"
	"github.com/urfave/cli"
)

var logger = log.NewWithModule("cmd")

func startCMD() cli.Command {
	return cli.Command{
		Name:  "start",
		Usage: "Start a long-running daemon process",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "config",
				Usage: "Specify BitXHub config path",
			},
			cli.StringFlag{
				Name:  "network",
				Usage: "Specify BitXHub network config path",
			},
			cli.StringFlag{
				Name:  "order",
				Usage: "Specify BitXHub order config path",
			},
			cli.StringFlag{
				Name:     "passwd",
				Usage:    "Specify BitXHub node private key password",
				Required: false,
			},
		},
		Action: start,
	}
}

func start(ctx *cli.Context) error {
	repoRoot, err := repo.PathRootWithDefault(ctx.GlobalString("repo"))
	if err != nil {
		return fmt.Errorf("get repo path: %w", err)
	}

	passwd := ctx.String("passwd")
	configPath := ctx.String("config")
	networkPath := ctx.String("network")
	orderPath := ctx.String("order")

	repo, err := repo.Load(repoRoot, passwd, configPath, networkPath)
	if err != nil {
		return fmt.Errorf("repo load: %w", err)
	}

	err = log.Initialize(
		log.WithReportCaller(repo.Config.Log.ReportCaller),
		log.WithPersist(true),
		log.WithFilePath(filepath.Join(repoRoot, repo.Config.Log.Dir)),
		log.WithFileName(repo.Config.Log.Filename),
		log.WithMaxAge(90*24*time.Hour),
		log.WithRotationTime(24*time.Hour),
	)
	if err != nil {
		return fmt.Errorf("log initialize: %w", err)
	}

	loggers.Initialize(repo.Config)

	types2.InitEIP155Signer(big.NewInt(int64(repo.Config.ChainID)))

	printVersion()

	if err := checkLicense(repo); err != nil {
		return fmt.Errorf("verify license fail:%v", err)
	}

	bxh, err := app.NewBitXHub(repo, orderPath)
	if err != nil {
		return fmt.Errorf("init bitxhub failed: %w", err)
	}

	monitor, err := profile.NewMonitor(repo.Config)
	if err != nil {
		return err
	}
	if err := monitor.Start(); err != nil {
		return err
	}

	pprof, err := profile.NewPprof(repo.Config)
	if err != nil {
		return err
	}
	if err := pprof.Start(); err != nil {
		return err
	}

	// coreapi
	api, err := coreapi.New(bxh)
	if err != nil {
		return err
	}

	// start grpc service
	b, err := grpc.NewChainBrokerService(api, repo.Config, &repo.Config.Genesis, bxh.Ledger)
	if err != nil {
		return err
	}

	if err := b.Start(); err != nil {
		return fmt.Errorf("start chain broker service failed: %w", err)
	}

	// start json-rpc service
	cbs, err := jsonrpc.NewChainBrokerService(api, repo.Config)
	if err != nil {
		return err
	}

	if err := cbs.Start(); err != nil {
		return fmt.Errorf("start chain broker service failed: %w", err)
	}

	gw := gateway.NewGateway(repo.Config)
	if err := gw.Start(); err != nil {
		fmt.Println(err)
	}

	bxh.Monitor = monitor
	bxh.Pprof = pprof
	bxh.Grpc = b
	bxh.Jsonrpc = cbs
	bxh.Gateway = gw

	var wg sync.WaitGroup
	wg.Add(1)
	handleLicenceCheck(bxh, repo, &wg)
	handleShutdown(bxh, &wg)

	if err := bxh.Start(); err != nil {
		return fmt.Errorf("start bitxhub failed: %w", err)
	}

	wg.Wait()

	return nil
}

func checkLicense(rep *repo.Repo) error {
	licenseCon, err := agency.GetLicenseConstructor("license")
	if err != nil {
		return nil
	}
	license := rep.Config.License
	licenseVerifier := licenseCon(license.Key, license.Verifier)
	return licenseVerifier.Verify(rep.Config.RepoRoot)
}

func printVersion() {
	fmt.Printf("BitXHub version: %s-%s-%s\n", bitxhub.CurrentVersion, bitxhub.CurrentBranch, bitxhub.CurrentCommit)
	fmt.Printf("App build date: %s\n", bitxhub.BuildDate)
	fmt.Printf("System version: %s\n", bitxhub.Platform)
	fmt.Printf("Golang version: %s\n", bitxhub.GoVersion)
	fmt.Println()
}

func handleLicenceCheck(node *app.BitXHub, repo *repo.Repo, wg *sync.WaitGroup) {
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := checkLicense(repo); err != nil {
					fmt.Printf("verify license fail:%v", err)
					if err := node.Stop(); err != nil {
						panic(err)
					}
					wg.Done()
					os.Exit(0)
				}
			}
		}
	}()
}
func handleShutdown(node *app.BitXHub, wg *sync.WaitGroup) {
	var stop = make(chan os.Signal)
	signal.Notify(stop, syscall.SIGTERM)
	signal.Notify(stop, syscall.SIGINT)

	go func() {
		<-stop
		fmt.Println("received interrupt signal, shutting down...")
		if err := node.Stop(); err != nil {
			panic(err)
		}
		wg.Done()
		os.Exit(0)
	}()
}
