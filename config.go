package main

import (
	"encoding/json"
	"os"
)

type Config struct {
	MattermostServer   string `json:"mattermostServer"`
	MattermostToken    string `json:"mattermostToken"`
	MattermostTeamName string `json:"mattermostTeamName"`
	MattermostChannel  string `json:"mattermostChannel"`
}

func loadConfig() *Config {
	file, err := os.Open("config.json")
	if err != nil {
		panic("Не удалось открыть config.json")
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	config := &Config{}
	err = decoder.Decode(config)
	if err != nil {
		panic("Ошибка при разборе config.json")
	}

	return config
}
