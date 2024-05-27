package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/walkerdu/wecom-backend/pkg/chatbot"
	"github.com/walkerdu/wecom-childrens-edu/configs"
	"github.com/walkerdu/wecom-childrens-edu/internal/pkg/handler"
	"github.com/walkerdu/wecom-childrens-edu/internal/pkg/service"

	"github.com/redis/go-redis/v9"
)

var (
	usage = `Usage: %s [options] [URL...]
Options:
	-f, --config_file <json config file>
`
	Usage = func() {
		//fmt.Println(fmt.Sprintf("Usage of %s:\n", os.Args[0]))
		fmt.Printf(usage, os.Args[0])
	}
)

func main() {
	log.SetFlags(log.Lshortfile | log.LstdFlags | log.Lmicroseconds)

	flag.Usage = Usage
	if len(os.Args) <= 1 {
		flag.Usage()
		os.Exit(1)
	}

	config := &configs.Config{}

	var configFile string
	flag.StringVar(&configFile, "f", "", "json config file")
	flag.StringVar(&configFile, "config_file", "", "json config file")

	flag.Parse()

	// 必须输入配置文件，
	if configFile == "" {
		flag.Usage()
		os.Exit(1)
	}

	//加载配置文件
	fileObj, err := os.Open(configFile)
	if err != nil {
		log.Fatalf("[ALERT] Open config file=%s failed, err=%s", configFile, err)
	}

	defer fileObj.Close()

	decoder := json.NewDecoder(fileObj)
	if err = decoder.Decode(config); err != nil {
		log.Fatalf("[ALERT] decode config file=%s failed, err=%s", configFile, err)
	}

	log.Printf("[INFO] starup config:%v", config)

	chatbot.NewChatbot(&chatbot.Config{OpenAI: config.OpenAI})

	// init redis client
	redisClient := initRedisClient(&config.Redis)
	if nil == redisClient {
		log.Fatalf("[ALERT] initRedisClient failed failed, addr=%v", config.Redis)
	}

	handler.HandlerInst().SetRedisClient(redisClient)

	ws, err := service.NewWeComServer(&config.WeCom)
	if err != nil {
		log.Fatal("[ALERT] NewWeComServer() failed")
	}

	// 优雅退出
	exitc := make(chan struct{})
	setupGracefulExitHook(exitc, ws)

	log.Printf("[INFO] start Serve()")
	ws.Serve()
}

func setupGracefulExitHook(exitc chan struct{}, ws *service.WeComServer) {
	log.Printf("[INFO] setupGracefulExitHook()")
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	go func() {
		sig := <-signalCh
		log.Printf("Got %s signal", sig)

		close(exitc)
		ws.Shutdown()
	}()
}

func initRedisClient(config *configs.RedisConfig) *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     config.Addr,
		Username: config.Username,
		Password: config.Password,
		DB:       config.DB,
	})

	return rdb
}
