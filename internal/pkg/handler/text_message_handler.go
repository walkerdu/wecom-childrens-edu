package handler

import (
	"context"
	"log"
	"strconv"
	"strings"
	"time"

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
			err = t.DBSet(textMsg)
			if err != nil {
				break
			}
			chatRsp = "success"
			break
		}

		switch content {
		case "/today":
			chatRsp, err = t.SummaryTodDay(textMsg)
		case "/month":
			chatRsp, err = t.SummaryMonth(textMsg)
		default:
			chatRsp, err = t.SummaryDay(textMsg)
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

func (t *TextMessageHandler) DBSet(msg *wecom.TextMessageReq) error {
	ctx := context.Background()
	date := time.Now().Format("20060102")
	key := msg.FromUserName + "_" + date
	result, err := HandlerInst().redisClient.RPush(ctx, key, msg.Content).Result()
	if err != nil {
		log.Printf("[ERROR][DBSet] redis LPush failed, err=%s", err)
		return err
	}

	log.Printf("[DEBUG][DBSet] redis LPush success, key:%v, value:%v, result=%v", key, msg.Content, result)
	return nil
}

func (t *TextMessageHandler) SummaryBase(ctx context.Context, key string) (string, error) {
	vals, err := HandlerInst().redisClient.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		log.Printf("[ERROR][SummaryBase] redis LRange failed, err=%s", err)
		return "", err
	}

	summary := ""
	for idx, val := range vals {
		summary += strconv.Itoa(idx+1) + ". " + val + "\n"
	}

	log.Printf("[DEBUG][SummaryBase] redis LRange success, key:%v, value:%v", key, vals)
	return summary, nil
}

func (t *TextMessageHandler) SummaryBaseBatch(ctx context.Context, keyPrefix string) (string, error) {
	keys, err := HandlerInst().redisClient.Keys(ctx, keyPrefix+"*").Result()
	if err != nil {
		log.Printf("[ERROR][SummaryMonth] redis Keys failed, err=%s", err)
		return "", err
	}

	summarys := ""
	for _, key := range keys {
		summary, err := t.SummaryBase(ctx, key)
		if err != nil {
			return "", err
		}

		summarys += key + "\n" + summary + "\n"
	}

	return summarys, nil
}

func (t *TextMessageHandler) SummaryTodDay(msg *wecom.TextMessageReq) (string, error) {
	ctx := context.Background()
	date := time.Now().Format("20060102")
	key := msg.FromUserName + "_" + date

	return t.SummaryBase(ctx, key)
}

func (t *TextMessageHandler) SummaryYesterday(msg *wecom.TextMessageReq) (string, error) {
	ctx := context.Background()

	date := time.Now().AddDate(0, 0, -1).Format("20060102")
	key := msg.FromUserName + "_" + date

	return t.SummaryBase(ctx, key)
}

func (t *TextMessageHandler) SummaryDay(msg *wecom.TextMessageReq) (string, error) {
	ctx := context.Background()

	content := strings.TrimSpace(msg.Content)
	key, found := strings.CutPrefix(content, "/")
	if !found {
		return "", nil
	}

	return t.SummaryBase(ctx, key)
}

// TODO 本周的前缀不一定相同
func (t *TextMessageHandler) SummaryWeek(msg *wecom.TextMessageReq) (string, error) {
	ctx := context.Background()
	date := time.Now().Format("200601")
	keyPrefix := msg.FromUserName + "_" + date

	return t.SummaryBaseBatch(ctx, keyPrefix)
}

func (t *TextMessageHandler) SummaryMonth(msg *wecom.TextMessageReq) (string, error) {
	ctx := context.Background()
	date := time.Now().Format("200601")
	keyPrefix := msg.FromUserName + "_" + date

	return t.SummaryBaseBatch(ctx, keyPrefix)
}

func (t *TextMessageHandler) Review() {
	ctx := context.Background()
	date := time.Now().Format("20060102")
	key := "walkerdu" + "_" + date

	summarys, err := t.SummaryBase(ctx, key)
	if err != nil {
		HandlerInst().publisher("walkerdu", "today data get failed, err:\n"+err.Error())
	}

	HandlerInst().publisher("walkerdu", "today:\n"+summarys)

	date = time.Now().Format("200601")
	keyPrefix := "walkerdu" + "_" + date

	summarys, err = t.SummaryBaseBatch(ctx, keyPrefix)
	if err != nil {
		HandlerInst().publisher("walkerdu", "month data get failed, err:\n"+err.Error())
	}

	HandlerInst().publisher("walkerdu", "month:\n"+summarys)
}
