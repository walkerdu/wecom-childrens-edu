package service

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/walkerdu/wecom-backend/pkg/chatbot"
	"github.com/walkerdu/wecom-backend/pkg/wecom"
	"github.com/walkerdu/wecom-childrens-edu/configs"
	"github.com/walkerdu/wecom-childrens-edu/internal/pkg/handler"
)

type WeComServer struct {
	httpSvr *http.Server
	wc      *wecom.WeCom
}

func NewWeComServer(config *configs.WeComConfig) (*WeComServer, error) {
	log.Printf("[INFO] NewWeComServer")

	svr := &WeComServer{}

	// 初始化企业微信API
	svr.wc = wecom.NewWeCom(&config.AgentConfig)

	mux := http.NewServeMux()
	mux.Handle("/wecom", svr.wc)
	mux.HandleFunc("/golds", svr.ServeHTTP)

	svr.httpSvr = &http.Server{
		Addr:    config.Addr,
		Handler: mux,
	}

	svr.initHandler()

	// 注册聊天消息的异步推送回调
	chatbot.MustChatbot().RegsiterMessagePublish(svr.wc.PushTextMessage)

	// 注册推送回调
	handler.HandlerInst().SetPublish(svr.wc.PushTextMessage)

	return svr, nil
}

// 注册企业微信消息处理的业务逻辑Handler
func (svr *WeComServer) initHandler() error {
	for msgType, handler := range handler.HandlerInst().GetLogicHandlerMap() {
		svr.wc.RegisterLogicMsgHandler(msgType, handler.HandleMessage)
	}

	return nil
}

func (svr *WeComServer) Serve() error {
	log.Printf("[INFO] Server()")

	if err := svr.httpSvr.ListenAndServe(); nil != err {
		log.Printf("httpSvr ListenAndServe() failed, err=%s", err)
		return err
	}

	return nil
}

func (svr *WeComServer) ReviewPubishing() {
	now := time.Now()
	awakeTime := time.Date(now.Year(), now.Month(), now.Day(), 23, 0, 0, 0, now.Location())

	if now.After(awakeTime) {
		awakeTime = awakeTime.Add(24 * time.Hour)
	}

	// 计算距离下次执行的时间间隔
	duration := awakeTime.Sub(now)

	// 创建一个定时器，在距离下次执行的时间间隔后触发
	timer := time.NewTimer(duration)

	for {
		select {
		case <-timer.C:
			log.Printf("[INFO] ReviewPubishing()")
			//txtHandler, _ := handler.HandlerInst().GetLogicHandler(wecom.MessageTypeText).(*handler.TextMessageHandler)
			//txtHandler.Review()
		}

		// 重新设置定时器，以实现每天定时执行
		timer.Reset(24 * time.Hour)
	}
}

func (svr *WeComServer) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := svr.httpSvr.Shutdown(ctx); err != nil {
		log.Printf("httpSvr ListenAndServe() failed, err=%s", err)
		return err
	}

	log.Println("[INFO]close httpSvr success")
	return nil
}

// ServeHTTP 实现http.Handler接口
func (svr *WeComServer) ServeHTTP(wr http.ResponseWriter, req *http.Request) {
	log.Printf("[DEBUG]ServeHttp|recv request URL:%s, Method:%s", req.URL, req.Method)

	if req.Method == http.MethodGet {
		// 解析请求的URI
		parsedURL := req.URL
		log.Printf("Path: %s\n", parsedURL.Path)
		log.Printf("RawQuery: %s\n", parsedURL.RawQuery)

		// 解析查询参数
		queryParams, err := url.ParseQuery(parsedURL.RawQuery)
		if err != nil {
			fmt.Printf("Failed to parse query parameters: %v\n", err)
			http.Error(wr, err.Error(), http.StatusBadRequest)
			return
		}

		var golds int64
		txtHandler, _ := handler.HandlerInst().GetLogicHandler(wecom.MessageTypeText).(*handler.TextMessageHandler)
		if _, exist := queryParams["incr"]; exist {
			golds, err = txtHandler.IncrGolds("duxingye")
		} else if _, exist := queryParams["duxingye"]; exist {
			golds, err = txtHandler.SummaryGolds("duxingye")
		}

		if err != nil {
			http.Error(wr, err.Error(), http.StatusBadRequest)
		} else {
			rsp_html := ` 
<!DOCTYPE html>
<html>
<head>
	<title>Number</title>
	<style>
		body {
			display: flex;
			flex-direction: column;
			justify-content: center;
			align-items: center;
			height: 100vh;
			margin: 0;
		}
		#container {
			display: flex;
			flex-direction: column;
			align-items: center;
		}
		#number {
			font-size: 48px;
			animation: blink 1s infinite;
			text-align: center;
		}
		@keyframes blink {
			0%% { color: blue; }
			50%% { color: red; }
			100%% { color: blue; }
		}
	</style>
</head>
<body>
	<div id="container">
		<img src="https://raw.githubusercontent.com/walkerdu/wecom-childrens-edu/master/assets/gold.png" alt="Gold Image">
		<div id="number">%d</div>
	</div>
</body>
</html>
`

			fmt.Fprintf(wr, rsp_html, golds)
		}

		return
	} else if req.Method == http.MethodPost {
		contentType := req.Header.Get("Content-Type")

		if contentType == "application/json" {
			// 1.http请求体body的content为json格式
			// 读取HTTP请求体
			body, err := ioutil.ReadAll(req.Body)
			if err != nil {
				err = fmt.Errorf("Failed to read request Body:%s", err)
				log.Printf("[DEBUG]%s", err)

				http.Error(wr, err.Error(), http.StatusInternalServerError)
				return
			}

			log.Printf("[DEBUG]ServeHTTP|recv request Body:%s", body)
		} else {
			err := fmt.Errorf("HTTP POST Method: unkown content-type:%s", contentType)
			log.Printf("[WARN]ServeHttp|%s", err)
			http.Error(wr, err.Error(), http.StatusBadRequest)

			return
		}

		fmt.Fprintf(wr, string(""))
	}
}
