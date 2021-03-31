package main

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/pprof"
	"sync"
	"syscall"
	"time"

	"github.com/meshplus/bitxhub"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub/api/gateway"
	"github.com/meshplus/bitxhub/api/grpc"
	"github.com/meshplus/bitxhub/api/jsonrpc"
	"github.com/meshplus/bitxhub/internal/app"
	"github.com/meshplus/bitxhub/internal/coreapi"
	"github.com/meshplus/bitxhub/internal/loggers"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/urfave/cli"
)

var logger = log.NewWithModule("cmd")

func startCMD() cli.Command {
	return cli.Command{
		Name:   "start",
		Usage:  "Start a long-running start process",
		Action: start,
	}
}

func start(ctx *cli.Context) error {
	repoRoot, err := repo.PathRootWithDefault(ctx.GlobalString("repo"))
	if err != nil {
		return fmt.Errorf("get repo path: %w", err)
	}

	repo, err := repo.Load(repoRoot)
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

	if repo.Config.PProf.Enable {
		switch repo.Config.PProf.PType {
		case "runtime":
			go runtimePProf(repo.Config.RepoRoot, repo.Config.PProf.Mode, repo.NetworkConfig.ID, repo.Config.PProf.Duration)
		case "http":
			httpPProf(repo.Config.Port.PProf)
		}
	}

	if repo.Config.Monitor.Enable {
		runMonitor(repo.Config.Port.Monitor)
	}

	printVersion()

	bxh, err := app.NewBitXHub(repo)
	if err != nil {
		return err
	}

	// coreapi
	api, err := coreapi.New(bxh)
	if err != nil {
		return err
	}

	// start grpc service
	b, err := grpc.NewChainBrokerService(api, repo.Config, &repo.Config.Genesis)
	if err != nil {
		return err
	}

	if err := b.Start(); err != nil {
		return err
	}

	// start json-rpc service
	cbs, err := jsonrpc.NewChainBrokerService(api, repo.Config)
	if err != nil {
		return err
	}

	if err := cbs.Start(); err != nil {
		return err
	}

	go func() {
		logger.WithField("port", repo.Config.Port.Gateway).Info("Gateway service started")
		err := gateway.Start(repo.Config)
		if err != nil {
			fmt.Println(err)
		}
	}()

	var wg sync.WaitGroup
	wg.Add(1)
	handleShutdown(bxh, &wg)

	if err := bxh.Start(); err != nil {
		return err
	}

	wg.Wait()

	return nil
}

func printVersion() {
	fmt.Printf("BitXHub version: %s-%s-%s\n", bitxhub.CurrentVersion, bitxhub.CurrentBranch, bitxhub.CurrentCommit)
	fmt.Printf("App build date: %s\n", bitxhub.BuildDate)
	fmt.Printf("System version: %s\n", bitxhub.Platform)
	fmt.Printf("Golang version: %s\n", bitxhub.GoVersion)
	fmt.Println()
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

// runtimePProf will record the cpu or memory profiles every 5 second.
func runtimePProf(repoRoot, mode string, id uint64, duration time.Duration) {
	tick := time.NewTicker(duration)
	rootPath := filepath.Join(repoRoot, "/pprof/")
	exist := fileExist(rootPath)
	if !exist {
		err := os.Mkdir(rootPath, os.ModePerm)
		if err != nil {
			fmt.Printf("----- runtimePProf start failed, err: %s -----\n", err.Error())
			return
		}
	}

	var cpuFile *os.File
	if mode == "cpu" {
		subPath := fmt.Sprint("cpu-", time.Now().Format("20060102-15:04:05"))
		cpuPath := filepath.Join(rootPath, subPath)
		cpuFile, _ = os.Create(cpuPath)
		_ = pprof.StartCPUProfile(cpuFile)
	}
	for {
		select {
		case <-tick.C:
			switch mode {
			case "cpu":
				pprof.StopCPUProfile()
				_ = cpuFile.Close()
				subPath := fmt.Sprint("cpu-", time.Now().Format("20060102-15:04:05"))
				cpuPath := filepath.Join(rootPath, subPath)
				cpuFile, _ := os.Create(cpuPath)
				_ = pprof.StartCPUProfile(cpuFile)
			case "memory":
				subPath := fmt.Sprint("mem-", time.Now().Format("20060102-15:04:05"))
				memPath := filepath.Join(rootPath, subPath)
				memFile, _ := os.Create(memPath)
				_ = pprof.WriteHeapProfile(memFile)
				_ = memFile.Close()
			}
		}
	}
}

func httpPProf(port int64) {
	go func() {
		addr := fmt.Sprintf(":%d", port)
		logger.WithField("port", port).Info("Start pprof")
		err := http.ListenAndServe(addr, nil)
		if err != nil {
			fmt.Println(err)
		}
	}()
}

// runMonitor runs prometheus handler
func runMonitor(port int64) {
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		addr := fmt.Sprintf(":%d", port)
		server := http.Server{
			Addr:    addr,
			Handler: mux,
		}
		logger.WithField("port", port).Info("Start monitor")
		err := server.ListenAndServe()
		if err != nil {
			fmt.Println(err)
		}
	}()
}

func fileExist(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}
