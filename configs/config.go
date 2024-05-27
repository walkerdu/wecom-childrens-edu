package configs

import (
	"github.com/walkerdu/wecom-backend/pkg/chatbot"
	"github.com/walkerdu/wecom-backend/pkg/wecom"
)

// 企业微信配置
type WeComConfig struct {
	AgentConfig wecom.AgentConfig `json:"agent_config"`
	Addr        string            `json:"addr"`
}

type RedisConfig struct {
	Addr     string `json:"addr"`
	Username string `json:"username"`
	Password string `json:"password"`
	DB       int    `json:"db"`
}

type Config struct {
	WeCom  WeComConfig          `json:"we_com"`
	OpenAI chatbot.OpenAIConfig `json:"open_ai"`
	Redis  RedisConfig          `json:"redis"`
}
