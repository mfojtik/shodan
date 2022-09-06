package config

import (
	"errors"
	"os"
	"strings"
)

type SlackConfig struct {
	AppToken string
	BotToken string
}

type Environment struct {
	Debug bool
	Slack *SlackConfig
}

func Read() (*Environment, error) {
	config := &Environment{}

	config.Debug = false
	if debugMode := os.Getenv("DEBUG_MODE"); len(debugMode) > 0 && strings.TrimSpace(debugMode) != "0" {
		config.Debug = true
	}

	var err error
	config.Slack, err = readSlackConfig()
	if err != nil {
		return nil, err
	}

	return config, nil
}

func readSlackConfig() (*SlackConfig, error) {
	config := &SlackConfig{}

	appToken := os.Getenv("SLACK_APP_TOKEN")
	if appToken == "" {
		return nil, errors.New("SLACK_APP_TOKEN must be set")
	}
	if !strings.HasPrefix(appToken, "xapp-") {
		return nil, errors.New("SLACK_APP_TOKEN must have the prefix \"xapp-\"")
	}
	config.AppToken = appToken

	botToken := os.Getenv("SLACK_BOT_TOKEN")
	if botToken == "" {
		return nil, errors.New("SLACK_BOT_TOKEN must be set")
	}
	if !strings.HasPrefix(botToken, "xoxb-") {
		return nil, errors.New("SLACK_BOT_TOKEN must have the prefix \"xoxb-\".")
	}
	config.BotToken = botToken

	return config, nil
}
