package livekit

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
	"io/ioutil"
	"math/rand"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"syscall"
	"time"

	"github.com/livekit/protocol/auth"

	"github.com/livekit/livekit-server/pkg/config"
	"github.com/livekit/livekit-server/pkg/logger"
	"github.com/livekit/livekit-server/pkg/routing"
	"github.com/livekit/livekit-server/pkg/service"
)

func init() {
	rand.Seed(time.Now().Unix())
}

//生成livekit的配置
func getConfig() (*config.Config, error) {
	confString, err := getConfigString()
	if err != nil {
		return nil, err
	}

	return config.NewConfig(confString, nil)
}

//启动livekit服务器
func Start() error {
	rand.Seed(time.Now().UnixNano())

	//生成cpu和mem的统计数据文件
	cpuProfile := ""
	memProfile := ""

	if cpuProfile != "" {
		if f, err := os.Create(cpuProfile); err != nil {
			return err
		} else {
			defer f.Close()
			if err := pprof.StartCPUProfile(f); err != nil {
				return err
			}
			defer pprof.StopCPUProfile()
		}
	}

	if memProfile != "" {
		if f, err := os.Create(memProfile); err != nil {
			return err
		} else {
			defer func() {
				// run memory profile at termination
				runtime.GC()
				_ = pprof.WriteHeapProfile(f)
				_ = f.Close()
			}()
		}
	}

	conf, err := getConfig()
	if err != nil {
		return err
	}

	if conf.Development {
		logger.InitDevelopment(conf.LogLevel)
	} else {
		logger.InitProduction(conf.LogLevel)
	}

	// require a key provider
	keyProvider, err := createKeyProvider(conf)
	if err != nil {
		return err
	}
	logger.Infow("configured key provider", "num_keys", keyProvider.NumKeys())

	currentNode, err := routing.NewLocalNode(conf)
	if err != nil {
		return err
	}

	// local routing and store
	router, roomStore, err := createRouterAndStore(conf, currentNode)
	if err != nil {
		return err
	}

	server, err := service.InitializeServer(conf, keyProvider,
		roomStore, router, currentNode, &routing.RandomSelector{})
	if err != nil {
		return err
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		sig := <-sigChan
		logger.Infow("exit requested, shutting down", "signal", sig)
		server.Stop()
	}()

	return server.Start()
}

//创建数据的临时存储，redis的v8和v9有冲突
func createRouterAndStore(config *config.Config, node routing.LocalNode) (router routing.Router, store service.RoomStore, err error) {
	if config.HasRedis() {
		logger.Infow("using multi-node routing via redis", "address", config.Redis.Address)
		rc := redis.NewClient(&redis.Options{
			Addr:     config.Redis.Address,
			Username: config.Redis.Username,
			Password: config.Redis.Password,
			DB:       config.Redis.DB,
		})
		if err = rc.Ping(context.Background()).Err(); err != nil {
			err = errors.Wrap(err, "unable to connect to redis")
			return
		}

		router = routing.NewRedisRouter(node, nil)
		store = service.NewRedisRoomStore(nil)
	} else {
		// local routing and store
		logger.Infow("using single-node routing")
		router = routing.NewLocalRouter(node)
		store = service.NewLocalRoomStore()
	}
	return
}

func createKeyProvider(conf *config.Config) (auth.KeyProvider, error) {
	// prefer keyfile if set
	if conf.KeyFile != "" {
		if st, err := os.Stat(conf.KeyFile); err != nil {
			return nil, err
		} else if st.Mode().Perm() != 0600 {
			return nil, fmt.Errorf("key file must have permission set to 600")
		}
		f, err := os.Open(conf.KeyFile)
		if err != nil {
			return nil, err
		}
		defer func() {
			_ = f.Close()
		}()
		return auth.NewFileBasedKeyProviderFromReader(f)
	}

	if len(conf.Keys) == 0 {
		return nil, errors.New("one of key-file or keys must be provided in order to support a secure installation")
	}

	return auth.NewFileBasedKeyProviderFromMap(conf.Keys), nil
}

//读取livekit的yaml配置文件
func getConfigString() (string, error) {
	configFile := "./conf/livekit.yaml"
	configBody := ""
	if configBody == "" {
		if configFile != "" {
			content, err := ioutil.ReadFile(configFile)
			if err != nil {
				return "", err
			}
			configBody = string(content)
		}
	}
	return configBody, nil
}
