package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mattermost/mattermost-server/v6/model"
)

func (app *application) createPoll(post *model.Post) {
	parts := strings.Split(post.Message, "\"")
	if len(parts) < 3 {
		app.sendMsgToChannel("Ошибка! Формат: @vote-bot голосование \"Вопрос\" \"Вариант1\" \"Вариант2\" ...", post.Id)
		return
	}

	question := parts[1]
	var options []string

	for i := 2; i < len(parts); i++ {
		option := strings.TrimSpace(parts[i])
		if option != "" {
			options = append(options, option)
		}
	}

	if len(options) < 2 {
		app.sendMsgToChannel("Ошибка! Нужно минимум два варианта ответа.", post.Id)
		return
	}

	pollID := fmt.Sprintf("poll-%d", time.Now().Unix())

	app.polls[pollID] = &Poll{
		ID:        pollID,
		Title:     question,
		Options:   options,
		Votes:     make(map[int]int),
		CreatedBy: post.UserId,
		Active:    true,
	}

	message := fmt.Sprintf("Голосование '%s' создано! ID: %s\nВарианты:\n", question, pollID)
	for i, option := range options {
		message += fmt.Sprintf("%d. %s\n", i+1, option)
	}

	app.sendMsgToChannel(message, post.Id)
}

func (app *application) vote(post *model.Post) {
	parts := strings.Split(post.Message, "\"")
	if len(parts) < 3 {
		app.sendMsgToChannel("Ошибка! Формат: @vote-bot голосовать \"ID\" \"номер_варианта\"", post.Id)
		return
	}

	pollID := parts[1]
	optionNumber := parts[3]

	optionIndex, err := strconv.Atoi(optionNumber)
	if err != nil || optionIndex < 1 {
		app.sendMsgToChannel("Ошибка! Неверный номер варианта.", post.Id)
		return
	}

	poll, exists := app.polls[pollID]
	if !exists || !poll.Active {
		app.sendMsgToChannel("Голосование с таким ID не найдено или уже завершено.", post.Id)
		return
	}

	if optionIndex > len(poll.Options) {
		app.sendMsgToChannel("Ошибка! Неверный номер варианта.", post.Id)
		return
	}

	poll.Votes[optionIndex]++
	app.sendMsgToChannel(fmt.Sprintf("Вы проголосовали за вариант %d: %s", optionIndex, poll.Options[optionIndex-1]), post.Id)
}

func (app *application) showResults(post *model.Post) {
	parts := strings.Split(post.Message, "\"")
	if len(parts) < 3 {
		app.sendMsgToChannel("Ошибка! Формат: @vote-bot результаты \"ID\"", post.Id)
		return
	}

	pollID := parts[1]
	poll, exists := app.polls[pollID]
	if !exists {
		app.sendMsgToChannel("Голосование с таким ID не найдено.", post.Id)
		return
	}

	result := fmt.Sprintf("Результаты голосования '%s' (ID: %s):\n", poll.Title, pollID)
	for i, option := range poll.Options {
		result += fmt.Sprintf("%d. %s: %d голосов\n", i+1, option, poll.Votes[i+1])
	}

	app.sendMsgToChannel(result, post.Id)
}

func (app *application) endPoll(post *model.Post) {
	parts := strings.Split(post.Message, "\"")
	if len(parts) < 3 {
		app.sendMsgToChannel("Ошибка! Формат: @vote-bot завершить \"ID\"", post.Id)
		return
	}

	pollID := parts[1]
	poll, exists := app.polls[pollID]
	if !exists || poll.CreatedBy != post.UserId {
		app.sendMsgToChannel("Вы не являетесь создателем этого голосования или оно не существует.", post.Id)
		return
	}

	poll.Active = false
	app.sendMsgToChannel(fmt.Sprintf("Голосование '%s' (ID: %s) завершено.", poll.Title, pollID), post.Id)
}

func (app *application) deletePoll(post *model.Post) {
	parts := strings.Split(post.Message, "\"")
	if len(parts) < 3 {
		app.sendMsgToChannel("Ошибка! Формат: @vote-bot удалить \"ID\"", post.Id)
		return
	}

	pollID := parts[1]
	_, exists := app.polls[pollID]
	if !exists {
		app.sendMsgToChannel("Голосование с таким ID не найдено.", post.Id)
		return
	}

	delete(app.polls, pollID)
	app.sendMsgToChannel(fmt.Sprintf("Голосование с ID: %s удалено.", pollID), post.Id)
}
