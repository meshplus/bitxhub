package main

import (
	"context"
	"fmt"
	"math/big"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/urfave/cli/v2"

	"github.com/axiomesh/axiom-kit/fileutil"
	types2 "github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom-ledger/api/jsonrpc"
	"github.com/axiomesh/axiom-ledger/internal/app"
	"github.com/axiomesh/axiom-ledger/internal/coreapi"
	"github.com/axiomesh/axiom-ledger/pkg/loggers"
	"github.com/axiomesh/axiom-ledger/pkg/profile"
	"github.com/axiomesh/axiom-ledger/pkg/repo"
)

func start(ctx *cli.Context) error {
	p, err := getRootPath(ctx)
	if err != nil {
		return err
	}

	if !fileutil.Exist(filepath.Join(p, repo.CfgFileName)) {
		fmt.Println("axiom-ledger is not initialized, please execute 'init.sh' first")
		return nil
	}

	r, err := repo.Load(p)
	if err != nil {
		return err
	}

	appCtx, cancel := context.WithCancel(ctx.Context)
	if err := loggers.Initialize(appCtx, r.Config); err != nil {
		cancel()
		return err
	}

	types2.InitEIP155Signer(big.NewInt(int64(r.Config.Genesis.ChainID)))

	printVersion()
	r.PrintNodeInfo()

	axm, err := app.NewAxiomLedger(r, appCtx, cancel)
	if err != nil {
		return fmt.Errorf("init axiom-ledger failed: %w", err)
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
		return fmt.Errorf("start axiom-ledger failed: %w", err)
	}

	if err := repo.WritePid(r.Config.RepoRoot); err != nil {
		return fmt.Errorf("write pid error: %s", err)
	}

	wg.Wait()

	if err := repo.RemovePID(r.Config.RepoRoot); err != nil {
		return fmt.Errorf("remove pid error: %s", err)
	}

	return nil
}

func printVersion() {
	fmt.Printf("%s version: %s-%s-%s\n", repo.AppName, repo.BuildVersion, repo.BuildBranch, repo.BuildCommit)
	fmt.Printf("App build date: %s\n", repo.BuildDate)
	fmt.Printf("System version: %s\n", repo.Platform)
	fmt.Printf("Golang version: %s\n", repo.GoVersion)
}

func handleShutdown(node *app.AxiomLedger, wg *sync.WaitGroup) {
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
