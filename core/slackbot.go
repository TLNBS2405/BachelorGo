package core

import (
	"fmt"
	"github.com/BachelorGo/responder"
	"github.com/nlopes/slack"
	"github.com/pkg/errors"
	"io/ioutil"
	"strings"
)

const slackTokenPath = "resources/slacktoken"

type SlackApp struct {
	slackToken     string
	client         *slack.Client
	rtm            *slack.RTM
	manager        *MessageManager
	conversationID string
}

func NewSlackBot(responder responder.Responder) (*SlackApp, error) {

	slackToken, err := getSlackAppToken(slackTokenPath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get slack token")
	}

	client := slack.New(slackToken)

	rtm := client.NewRTM()
	manager, err := NewMessageManager(responder)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create MessageManager")
	}

	return &SlackApp{slackToken: slackToken, client: client, rtm: rtm, manager: manager, conversationID: "1"}, nil
}

func (slackApp *SlackApp) Run() {

	go slackApp.rtm.ManageConnection()
	for {
		select {
		case message := <-slackApp.rtm.IncomingEvents:
			switch event := message.Data.(type) {
			case *slack.ConnectedEvent:
				fmt.Println("Connection counter:", event.ConnectionCount)

			case *slack.MessageEvent:
				fmt.Printf("Message: %v\n", event.Text)
				slackApp.Respond(event)

			case *slack.RTMError:
				fmt.Printf("Error: %s\n", event.Error())

			case *slack.InvalidAuthEvent:
				fmt.Printf("Invalid credentials. Exiting")
				break

			}
		}
	}

}

func (slackApp *SlackApp) Respond(msg *slack.MessageEvent) {

	response := ""
	text := msg.Text

	if strings.ToLower(text) == "%new" {

		slackApp.conversationID = slackApp.getNewConversationID()
		response = "new conversation with ID:" + slackApp.conversationID

		newManager, err := NewMessageManager(slackApp.manager.responder)
		if err != nil {
			response = "An error occurred: " + err.Error()
		}
		slackApp.manager = newManager

		slackApp.rtm.SendMessage(slackApp.rtm.NewOutgoingMessage(response, msg.Channel))

		return

	} else if strings.Contains(strings.ToLower(text), "%switch") {

		slackApp.conversationID = slackApp.getConversationID(text)
		response = "switch to conversation with ID:" + slackApp.conversationID

		slackApp.rtm.SendMessage(slackApp.rtm.NewOutgoingMessage(response, msg.Channel))
		return
	}

	response, err := slackApp.manager.Response(text, slackApp.conversationID)
	if err != nil {
		response = "An error occurred: " + err.Error()
	}

	slackApp.rtm.SendMessage(slackApp.rtm.NewOutgoingMessage(response, msg.Channel))
}

func (slackApp *SlackApp) getNewConversationID() string {
	newID := slackApp.manager.NewRandomConversationID()
	return newID
}

func (slackApp *SlackApp) getConversationID(text string) string {
	values := strings.Split(text, " ")
	convID := strings.TrimSpace(values[1])
	return convID
}

func getSlackAppToken(path string) (string, error) {

	file, err := ioutil.ReadFile(path)
	if err != nil {
		return "", errors.Wrapf(err, "failed to open slack ooken file")
	}

	result := string(file)

	return result, nil
}
