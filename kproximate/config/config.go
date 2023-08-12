package config

import (
	"context"
	"regexp"

	"github.com/Telmate/proxmox-api-go/proxmox"
	"github.com/sethvargo/go-envconfig"
)

type KproximateConfig struct {
	KpNodeCores        int     `env:"kpNodeCores"`
	KpLoadHeadroom     float64 `env:"kpLoadHeadroom"`
	KpNodeMemory       int     `env:"kpNodeMemory"`
	KpNodeNamePrefix   string  `env:"kpNodeNamePrefix"`
	KpNodeNameRegex    *regexp.Regexp
	KpNodeParams       map[string]interface{}
	KpNodeTemplateName string `env:"kpNodeTemplateName"`
	KpNodeTemplateRef  proxmox.VmRef
	MaxKpNodes         int    `env:"maxKpNodes"`
	PmAllowInsecure    bool   `env:"pmAllowInsecure"`
	PmDebug            bool   `env:"pmDebug"`
	PmToken            string `env:"pmToken"`
	PmUrl              string `env:"pmUrl"`
	PmUserID           string `env:"pmUserID"`
	PollInterval       int    `env:"pollInterval"`
	SshKey             string `env:"sshKey"`
	WaitSecondsForJoin int    `env:"waitSecondsForJoin"`
}

type RabbitConfig struct {
	Host     string `env:"rabbitMQHost"`
	Password string `env:"rabbitMQPassword"`
	Port     int    `env:"rabbitMQPort"`
	User     string `env:"rabbitMQUser"`
}

func GetKpConfig() (KproximateConfig, error) {
	config := &KproximateConfig{}

	err := envconfig.Process(context.Background(), config)
	if err != nil {
		return *config, err
	}

	*config = validateConfig(config)

	return *config, nil
}

func GetRabbitConfig() (RabbitConfig, error) {
	config := &RabbitConfig{}

	err := envconfig.Process(context.Background(), config)
	if err != nil {
		return *config, err
	}

	return *config, nil
}

func validateConfig(config *KproximateConfig) KproximateConfig {
	if config.KpLoadHeadroom < 0.2 {
		config.KpLoadHeadroom = 0.2
	}

	if config.PollInterval < 10 {
		config.PollInterval = 10
	}

	if config.WaitSecondsForJoin < 60 {
		config.WaitSecondsForJoin = 60
	}
	return *config
}
