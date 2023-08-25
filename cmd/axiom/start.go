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

	"github.com/urfave/cli/v2"

	"github.com/axiomesh/axiom"
	"github.com/axiomesh/axiom-kit/fileutil"
	"github.com/axiomesh/axiom-kit/log"
	types2 "github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom/api/jsonrpc"
	"github.com/axiomesh/axiom/internal/app"
	"github.com/axiomesh/axiom/internal/coreapi"
	"github.com/axiomesh/axiom/pkg/loggers"
	"github.com/axiomesh/axiom/pkg/profile"
	"github.com/axiomesh/axiom/pkg/repo"
)

const (
	full          = "full"
	mockConsensus = "mockConsensus"
	mockExecutor  = "mockExecutor"
)

var startCMD = &cli.Command{
	Name:  "start",
	Usage: "Start a long-running daemon process",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "mode",
			Usage:    "start axiom in specified mode",
			Required: false,
			Value:    full,
		},
	},
	Action: start,
}

func start(ctx *cli.Context) error {
	p, err := getRootPath(ctx)
	if err != nil {
		return err
	}
	existConfig := fileutil.Exist(p)
	r, err := repo.Load(p)
	if err != nil {
		return err
	}

	if !existConfig {
		// not generate config, start by solo
		r.Config.Order.Type = repo.OrderTypeSolo
		if err := r.Flush(); err != nil {
			return err
		}
	}
	mode := ctx.String("mode")
	if mode == mockConsensus {
		r.Config.Order.Type = repo.OrderTypeSoloDev
	}

	if mode == mockExecutor {
		r.Config.Executor.Type = repo.ExecTypeDev
	}

	err = log.Initialize(
		log.WithReportCaller(r.Config.Log.ReportCaller),
		log.WithPersist(true),
		log.WithFilePath(filepath.Join(r.Config.RepoRoot, repo.LogsDirName)),
		log.WithFileName(r.Config.Log.Filename),
		log.WithMaxAge(r.Config.Log.MaxAge.ToDuration()),
		log.WithRotationTime(r.Config.Log.RotationTime.ToDuration()),
	)
	if err != nil {
		return fmt.Errorf("log initialize: %w", err)
	}

	loggers.Initialize(r.Config)

	types2.InitEIP155Signer(big.NewInt(int64(r.Config.Genesis.ChainID)))

	printVersion()

	axm, err := app.NewAxiom(r)
	if err != nil {
		return fmt.Errorf("init axiom failed: %w", err)
	}

	monitor, err := profile.NewMonitor(r.Config)
	if err != nil {
		return err
	}
	if err := monitor.Start(); err != nil {
		return err
	}

	pprof, err := profile.NewPprof(r.Config)
	if err != nil {
		return err
	}
	if err := pprof.Start(); err != nil {
		return err
	}

	// coreapi
	api, err := coreapi.New(axm)
	if err != nil {
		return err
	}

	// start json-rpc service
	cbs, err := jsonrpc.NewChainBrokerService(api, r.Config)
	if err != nil {
		return err
	}

	if err := cbs.Start(); err != nil {
		return fmt.Errorf("start chain broker service failed: %w", err)
	}

	axm.Monitor = monitor
	axm.Pprof = pprof
	axm.Jsonrpc = cbs

	var wg sync.WaitGroup
	wg.Add(1)
	handleShutdown(axm, &wg)

	if err := axm.Start(); err != nil {
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
