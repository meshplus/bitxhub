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

	"github.com/axiomesh/axiom"
	"github.com/axiomesh/axiom-kit/log"
	types2 "github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom/api/jsonrpc"
	"github.com/axiomesh/axiom/internal/app"
	"github.com/axiomesh/axiom/internal/coreapi"
	"github.com/axiomesh/axiom/internal/loggers"
	"github.com/axiomesh/axiom/internal/profile"
	"github.com/axiomesh/axiom/internal/repo"
	"github.com/urfave/cli"
)

func startCMD() cli.Command {
	return cli.Command{
		Name:  "start",
		Usage: "Start a long-running daemon process",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "config",
				Usage: "Specify Axiom config path",
			},
			cli.StringFlag{
				Name:  "network",
				Usage: "Specify Axiom network config path",
			},
			cli.StringFlag{
				Name:  "order",
				Usage: "Specify Axiom order config path",
			},
			cli.StringFlag{
				Name:     "passwd",
				Usage:    "Specify Axiom node private key password",
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

	types2.InitEIP155Signer(big.NewInt(int64(repo.Config.Genesis.ChainID)))

	printVersion()

	bxh, err := app.NewAxiom(repo, orderPath)
	if err != nil {
		return fmt.Errorf("init axiom failed: %w", err)
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

	// start json-rpc service
	cbs, err := jsonrpc.NewChainBrokerService(api, repo.Config)
	if err != nil {
		return err
	}

	if err := cbs.Start(); err != nil {
		return fmt.Errorf("start chain broker service failed: %w", err)
	}

	bxh.Monitor = monitor
	bxh.Pprof = pprof
	bxh.Jsonrpc = cbs

	var wg sync.WaitGroup
	wg.Add(1)
	handleShutdown(bxh, &wg)

	if err := bxh.Start(); err != nil {
		return fmt.Errorf("start axiom failed: %w", err)
	}

	wg.Wait()

	return nil
}

func printVersion() {
	fmt.Printf("Axiom version: %s-%s-%s\n", axiom.CurrentVersion, axiom.CurrentBranch, axiom.CurrentCommit)
	fmt.Printf("App build date: %s\n", axiom.BuildDate)
	fmt.Printf("System version: %s\n", axiom.Platform)
	fmt.Printf("Golang version: %s\n", axiom.GoVersion)
	fmt.Println()
}

func handleShutdown(node *app.Axiom, wg *sync.WaitGroup) {
	var stop = make(chan os.Signal, 2)
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
