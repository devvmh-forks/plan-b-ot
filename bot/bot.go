// This file is part of the Plan-B-ot package.
// Copyright (c) 2015 Martin Schenck
// For the full copyright and license information, please view the LICENSE
// file that was distributed with this source code.

package bot

import (
	"bytes"
	"fmt"
	"net/http"
	"strconv"
)

type CommandResponse struct {
	ResponseType string      `json:"response_type"`
	Text         string      `json:"text"`
	GotoLocation string      `json:"goto_location"`
	Attachments  interface{} `json:"attachments"`
}

// HandleRequest handles slash commands coming from slack
func HandleRequest(userName string, args []string) (response CommandResponse, status int) {
	if len(args) < 1 {
		return CommandResponse{Text: "Please define something to do afte the slash command. E.g. `/planbot task T54`"}, http.StatusBadRequest
	}
	action := args[0]

	responseText := "Unknown Error"
	status = http.StatusInternalServerError

	if action == "task" {
		responseText, status = setTask(userName, args)
	} else if action == "vote" {
		responseText, status = setVote(userName, args)
	} else if action == "results" {
		responseText, status = getResults()
	} else {
		responseText = fmt.Sprintf("Invalid item after the slash command: `%s`.", action)
		status = http.StatusBadRequest
	}

	response = CommandResponse{
		ResponseType: "ephemeral",
		Text:         responseText,
	}

	return response, status
}

// setTask sets a new task witha  given name
func setTask(userName string, args []string) (response string, status int) {
	if len(args) < 2 {
		return "Please specify the task name. E.g. `/planbot task T54`", http.StatusBadRequest
	}
	taskName := args[1]

	currentTask = task{
		Name:  taskName,
		Votes: make([]vote, 0),
	}

	err := sendToSlack(
		"Task set",
		fmt.Sprintf("@%s set the task to %s. All votes have been reset.", userName, taskName),
		"good",
	)
	if err != nil {
		return err.Error(), http.StatusInternalServerError
	}

	return fmt.Sprintf("You set the task to %s. All votes have been reset.", taskName), http.StatusOK
}

// setVote adds a vote to the current task
func setVote(userName string, args []string) (response string, status int) {
	if currentTask.Name == "" {
		return "No task set. Please specify a task first. E.g. `/planbot task T65`", http.StatusBadRequest
	}

	if len(args) < 2 {
		return "Please specify the vote value. E.g. `/planbot vote 3`", http.StatusBadRequest
	}
	storyPoints := args[1]

	err := currentTask.Vote(userName, storyPoints)
	if err != nil {
		return "Please specify a float value for points. E.g. `/planbot vote 3`", http.StatusBadRequest
	}

	err = sendToSlack(
		"Vote received",
		fmt.Sprintf("@%s voted on task `%s`.\nVoters: %s", userName, currentTask.Name, currentTask.getVoters()),
		"good",
	)
	if err != nil {
		return err.Error(), http.StatusInternalServerError
	}

	return fmt.Sprintf("You voted `%s` story points on task `%s`", storyPoints, currentTask.Name), http.StatusOK
}

// getResult reurns the current votes and resets the task
// The task stays with zero votes until a new one is set with setTask()
func getResults() (response string, status int) {
	if currentTask.Name == "" {
		return "No task set. Please specify a task first. E.g. `/planbot task T65`", http.StatusBadRequest
	}

	if len(currentTask.Votes) == 0 {
		return fmt.Sprintf("No votes on current task %s", currentTask.Name), http.StatusOK
	}

	var buffer bytes.Buffer

        max := 0.0
        min := 9999.0
	for _, vote := range currentTask.Votes {
		storyPoints := strconv.FormatFloat(vote.Vote, 'f', -1, 64)
                if (max < vote.Vote) {
                  max = vote.Vote
                }
                if (min > vote.Vote) {
                  min = vote.Vote
                }

		buffer.WriteString("@")
		buffer.WriteString(vote.Username)
		buffer.WriteString(": `")
		buffer.WriteString(storyPoints)
		buffer.WriteString("`\n")
	}

	buffer.WriteString("Max: `")
	buffer.WriteString(strconv.FormatFloat(max, 'g', -1, 64))
	buffer.WriteString("`\n")

	buffer.WriteString("Min: `")
	buffer.WriteString(strconv.FormatFloat(min, 'g', -1, 64))
	buffer.WriteString("`\n")

	currentTask = task{
		Name:  currentTask.Name,
		Votes: make([]vote, 0),
	}

	err := sendToSlack(
		"Results for task "+currentTask.Name,
		buffer.String(),
		"good",
	)
	if err != nil {
		return err.Error(), http.StatusInternalServerError
	}
	return "Results were printed in channel", http.StatusOK
}
