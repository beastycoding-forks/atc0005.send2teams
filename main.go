// Copyright 2019 Adam Chalkley
//
// https://github.com/atc0005/send2teams
//
// Licensed under the MIT License. See LICENSE file in the project root for
// full license information.

package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	goteamsnotify "gopkg.in/dasrick/go-teams-notify.v1"
)

// All webhook URLs begin with this URL pattern. We check provided URLs against
// this pattern.
const webhookURLPrefix = "https://outlook.office.com/webhook/"

const defaultMessageThemeColor = "#832561"

// TeamsChannel represents a Microsoft Teams channel that this application
// will attempt to submit messages to.
type TeamsChannel struct {

	// Team is the human-readable name of the Microsoft Teams "team" that
	// contains the channel we wish to post a message to. This is used in
	// informational output produced by this application.
	Team string

	// Channel is human-readable name of the channel within a specific
	// Microsoft Teams "team". This is used in informational output produced
	// by this application.
	Channel string

	// WebhookURL is the full URL used to submit messages to the Teams channel
	// This URL is in the form of https://outlook.office.com/webhook/xxx This
	// URL is REQUIRED in order for this application to function and needs to
	// be created in advance by adding/configuring a Webhook Connector in a
	// Microsoft Teams channel that you wish to submit messages to using this
	// application.
	WebhookURL string
}

// TeamsMessage represents a message that this application will attempt to
// submit to the specified Microsoft Teams Channel.
type TeamsMessage struct {

	// ThemeColor is a hex color code string representing the desired border
	// trim color for our submitted messages.
	ThemeColor string

	// MessageTitle is the text shown on the top portion of the message "card"
	// that is displayed in Microsoft Teams for the message that we send.
	MessageTitle string

	// MessageText is an (optionally) Markdown-formatted string representing the
	// message that we will submit.
	MessageText string
}

func validateWebhook(webhook TeamsChannel) error {

	// ensure that all fields meet at least the minimum basic requirements

	if len(webhook.Team) == 0 {
		return fmt.Errorf("team name too short")

	}

	if len(webhook.Channel) == 0 {
		return fmt.Errorf("channel name too short")
	}

	// Ensure that at least the prefix is present
	// TODO: Is the URL pattern stable? If so, perhaps we can add a length
	// check also?
	if !strings.HasPrefix(webhook.WebhookURL, webhookURLPrefix) {
		webhookURLPrefixLength := len(webhookURLPrefix)
		actualWebhookURLPrefix := webhook.WebhookURL[0:(webhookURLPrefixLength - 1)]
		return fmt.Errorf("webhook URL missing expected prefix! Got: %q, Expected: %q",
			actualWebhookURLPrefix, webhook.WebhookURL)
	}

	// Indicate that we didn't spot any problems
	return nil

}

func validateMessage(message TeamsMessage) error {

	// ensure that all fields meet basic requirements

	// Expected pattern: #832561
	if len(message.ThemeColor) < len(defaultMessageThemeColor) {

		expectedLength := len(defaultMessageThemeColor)
		actualLength := len(message.ThemeColor)
		return fmt.Errorf("Provided message theme color too short. Got message %q of length %d, expected length of %d",
			message.ThemeColor, actualLength, expectedLength)
	}

	if message.MessageTitle == "" {
		return fmt.Errorf("message title too short")
	}

	if message.MessageText == "" {
		return fmt.Errorf("message content too short")
	}

	// Indicate that we didn't spot any problems
	return nil

}

func sendMessage(webhookURL string, teamsMessage goteamsnotify.MessageCard) error {

	// init the client
	mstClient, err := goteamsnotify.NewClient()
	if err != nil {
		return err
	}

	// attempt to send message, return the pass/fail result to caller
	return mstClient.Send(webhookURL, teamsMessage)
}

func main() {

	webhook := TeamsChannel{}
	message := TeamsMessage{}

	flag.StringVar(&webhook.Team, "team", "", "The name of the Team containing our target channel")
	flag.StringVar(&webhook.Channel, "channel", "", "The target channel where we will send a message")
	flag.StringVar(&webhook.WebhookURL, "url", "", "The Webhook URL provided by a preconfigured Connector")
	flag.StringVar(&message.ThemeColor, "color", "", "The hex color code used to set the desired trim color on submitted messages")
	flag.StringVar(&message.MessageTitle, "title", "", "The title for the message to submit")
	flag.StringVar(&message.MessageText, "message", "", "The (optionally) Markdown-formatted message to submit")

	// parse flag definitions from the argument list
	flag.Parse()

	// If the theme color option wasn't provided, set it to a default value.
	if message.ThemeColor == "" {
		message.ThemeColor = defaultMessageThemeColor
	}

	// Validate provided info before going any further
	if err := validateMessage(message); err != nil {
		log.Fatal(err)

	}

	if err := validateWebhook(webhook); err != nil {
		log.Fatal(err)
	}

	// setup message card
	msgCard := goteamsnotify.NewMessageCard()
	msgCard.Title = message.MessageTitle
	msgCard.Text = message.MessageText
	msgCard.ThemeColor = message.ThemeColor

	if err := sendMessage(webhook.WebhookURL, msgCard); err != nil {
		log.Fatalf("Failed to submit message to %q channel in the %q team!\nMessage: %+v\nError: %v",
			webhook.Channel, webhook.Team, message, err)
	}

	fmt.Printf("Message successfully sent to %q channel in the %q team:\n%+v\n",
		webhook.Channel, webhook.Team, message)

}
