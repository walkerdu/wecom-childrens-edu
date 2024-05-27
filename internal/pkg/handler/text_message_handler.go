package handler

import (
	"context"
	"errors"
	"log"
	"strconv"
	"strings"

	//"github.com/walkerdu/wecom-backend/pkg/chatbot"
	"github.com/walkerdu/wecom-backend/pkg/wecom"
)

const WeChatTimeOutSecs = 5

func init() {
	handler := &TextMessageHandler{}

	HandlerInst().RegisterLogicHandler(wecom.MessageTypeText, handler)
}

type TextMessageHandler struct {
}

var commandsMap = map[string]struct{}{
	"/today":     struct{}{},
	"/yesterday": struct{}{},
	"/week":      struct{}{},
	"/month":     struct{}{},
}

func (t *TextMessageHandler) GetHandlerType() wecom.MessageType {
	return wecom.MessageTypeText
}

func (t *TextMessageHandler) HandleMessage(msg wecom.MessageIF) (wecom.MessageIF, error) {
	textMsg := msg.(*wecom.TextMessageReq)

	var chatRsp string
	var err error
	content := strings.TrimSpace(textMsg.Content)

	for {
		if !strings.HasPrefix(content, "/") {
			err = errors.New("unknow command")
			break
		}

		switch content {
		case "/杜行烨":
			var golds int64
			golds, err = t.SummaryGolds("duxingye")
			chatRsp = string(golds)
		case "/杜行逸":
			var golds int64
			golds, err = t.SummaryGolds("duxingyi")
			chatRsp = string(golds)
		case "/reset":
			err = t.ResetGolds("duxingye")
			chatRsp = "杜行烨数据清零"
		default:
			err = errors.New("unknow command")
		}

		// 指令请求，保证无数据也返回消息
		if chatRsp == "" && err == nil {
			chatRsp = "no data"
		}

		break
	}

	if err != nil {
		chatRsp = err.Error()
	}

	//chatRsp, err := chatbot.MustChatbot().GetResponse(textMsg.FromUserName, textMsg.Content)
	//if err != nil {
	//	log.Printf("[ERROR][HandleMessage] chatbot.GetResponse failed, err=%s", err)
	//	chatRsp = "chatbot something wrong, errMsg:" + err.Error()
	//}

	textMsgRsp := wecom.TextMessageRsp{
		Content: chatRsp,
	}

	return &textMsgRsp, nil
}

func (t *TextMessageHandler) IncrGolds(key string) (int64, error) {
	ctx := context.Background()
	key += "_golds"
	result, err := HandlerInst().redisClient.Incr(ctx, key).Result()
	if err != nil {
		log.Printf("[ERROR][DBSet] redis LPush failed, err=%s", err)
		return 0, err
	}

	log.Printf("[DEBUG][DBSet] redis Incr success, key:%v, after value:%v", key, result)
	return result, nil
}

func (t *TextMessageHandler) SummaryGolds(key string) (int64, error) {
	ctx := context.Background()
	key += "_golds"

	val, err := HandlerInst().redisClient.Get(ctx, key).Result()
	if err != nil {
		log.Printf("[ERROR][SummaryBase] redis Get failed, err=%s", err)
		return 0, err
	}

	log.Printf("[DEBUG][SummaryBase] redis Get success, key:%v, value:%v", key, val)
	golds, err := strconv.Atoi(val)
	if err != nil {
		log.Printf("[ERROR][SummaryBase] redis Get golds invalid, err=%s", err)
		return 0, err
	}

	return int64(golds), nil
}

func (t *TextMessageHandler) ResetGolds(key string) error {
	ctx := context.Background()
	key += "_golds"

	val, err := HandlerInst().redisClient.Del(ctx, key).Result()
	if err != nil {
		log.Printf("[ERROR][SummaryBase] redis Get failed, err=%s", err)
		return err
	}

	log.Printf("[DEBUG][SummaryBase] redis Get success, key:%v, value:%v", key, val)
	return nil
}
