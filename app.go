package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/rs/zerolog"
)

type Poll struct {
	ID        string
	Title     string
	Options   []string
	Votes     map[int]int
	CreatedBy string
	Active    bool
}

type application struct {
	logger                    zerolog.Logger
	config                    *Config
	mattermostClient          *model.Client4
	mattermostWebSocketClient *model.WebSocketClient
	mattermostUser            *model.User
	mattermostTeam            *model.Team
	mattermostChannel         *model.Channel
	polls                     map[string]*Poll
}

func (app *application) init() {
	app.config = loadConfig()
	app.logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC822}).With().Timestamp().Logger()

	app.mattermostClient = model.NewAPIv4Client(app.config.MattermostServer)
	app.mattermostClient.SetToken(app.config.MattermostToken)

	if user, _, err := app.mattermostClient.GetUser("me", ""); err != nil {
		app.logger.Fatal().Err(err).Msg("Не удалось войти в систему")
	} else {
		app.mattermostUser = user
	}

	if team, _, err := app.mattermostClient.GetTeamByName(app.config.MattermostTeamName, ""); err != nil {
		app.logger.Fatal().Err(err).Msg("Не удалось найти команду")
	} else {
		app.mattermostTeam = team
	}

	if channel, _, err := app.mattermostClient.GetChannelByName(app.config.MattermostChannel, app.mattermostTeam.Id, ""); err != nil {
		app.logger.Fatal().Err(err).Msg("Не удалось найти канал")
	} else {
		app.mattermostChannel = channel
	}

	app.polls = make(map[string]*Poll)

	setupGracefulShutdown(app)
}

func (app *application) startWebSocket() {
	var err error
	for {
		app.mattermostWebSocketClient, err = model.NewWebSocketClient4(
			fmt.Sprintf("ws://%s", app.config.MattermostServer[7:]),
			app.mattermostClient.AuthToken,
		)
		if err != nil {
			app.logger.Warn().Err(err).Msg("Ошибка WebSocket, пробуем снова через 5 секунд")
			time.Sleep(5 * time.Second)
			continue
		}
		app.logger.Info().Msg("WebSocket подключен")
		app.mattermostWebSocketClient.Listen()

		for event := range app.mattermostWebSocketClient.EventChannel {
			go app.handleWebSocketEvent(event)
		}
	}
}

func (app *application) handleWebSocketEvent(event *model.WebSocketEvent) {
	if event.GetBroadcast().ChannelId != app.mattermostChannel.Id {
		return
	}
	if event.EventType() != model.WebsocketEventPosted {
		return
	}

	post := &model.Post{}
	err := json.Unmarshal([]byte(event.GetData()["post"].(string)), &post)
	if err != nil {
		app.logger.Error().Err(err).Msg("Ошибка обработки сообщения")
		return
	}

	if post.UserId == app.mattermostUser.Id {
		return
	}

	app.handlePost(post)
}

func (app *application) handlePost(post *model.Post) {
	if strings.HasPrefix(post.Message, "@vote-bot голосование") {
		app.createPoll(post)
	} else if strings.HasPrefix(post.Message, "@vote-bot голосовать") {
		app.vote(post)
	} else if strings.HasPrefix(post.Message, "@vote-bot результаты") {
		app.showResults(post)
	} else if strings.HasPrefix(post.Message, "@vote-bot завершить") {
		app.endPoll(post)
	} else if strings.HasPrefix(post.Message, "@vote-bot удалить") {
		app.deletePoll(post)
	}
}

func (app *application) sendMsgToChannel(message string, postId string) {
	post := &model.Post{
		ChannelId: app.mattermostChannel.Id,
		Message:   message,
		RootId:    postId,
	}

	_, _, err := app.mattermostClient.CreatePost(post)
	if err != nil {
		app.logger.Error().Err(err).Msg("Ошибка при отправке сообщения в канал")
	}
}

func setupGracefulShutdown(app *application) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			if app.mattermostWebSocketClient != nil {
				app.logger.Info().Msg("Закрываем WebSocket")
				app.mattermostWebSocketClient.Close()
			}
			app.logger.Info().Msg("Выключаем бота")
			os.Exit(0)
		}
	}()
}
