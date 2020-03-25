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
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"regexp"
	"strings"

	//goteamsnotify "gopkg.in/dasrick/go-teams-notify.v1"

	// temporarily use our fork until upstream webhook URL FQDN validation
	// changes can be made
	goteamsnotify "github.com/atc0005/go-teams-notify"
)

// Overridden via Makefile for release builds
var version string = "dev build"

// Primarily used with branding
const myAppName string = "send2teams"
const myAppURL string = "https://github.com/atc0005/send2teams"

// In practice, all new webhook URLs appear to use the outlook.office.com
// FQDN. However, some older guides, and even the current official
// documentation, use outlook.office365.com in their webhook URL examples.
// https://docs.microsoft.com/en-us/outlook/actionable-messages/send-via-connectors
const webhookURLOfficecomPrefix = "https://outlook.office.com"
const webhookURLOffice365Prefix = "https://outlook.office365.com"
const webhookURLOfficialDocsSampleURI = "webhook/a1269812-6d10-44b1-abc5-b84f93580ba0@9e7b80c7-d1eb-4b52-8582-76f921e416d9/IncomingWebhook/3fdd6767bae44ac58e5995547d66a4e4/f332c8d9-3397-4ac5-957b-b8e3fc465a8c"

// Build a regular expression that we can use to validate incoming webhook
// URLs provided by the user.
//
// Note: The regex allows for capital letters in the GUID patterns. This is
// allowed based on light testing which shows that mixed case works and the
// assumption that since Teams and Office 365 are Microsoft products case
// would be ignored (e.g., Windows, IIS do not consider 'A' and 'a' to be
// different).
var validWebhookURLRegex = `^https:\/\/outlook.office(?:365)?.com\/webhook\/[-a-zA-Z0-9]{36}@[-a-zA-Z0-9]{36}\/IncomingWebhook\/[-a-zA-Z0-9]{32}\/[-a-zA-Z0-9]{36}$`

// Used if the user doesn't provide a value via commandline
const defaultMessageThemeColor = "#832561"

// TODO: Why is the double leading slash necessary to match on escape
// sequences in order to replacement them?

// Used by Teams to separate lines
const breakStatement = "<br>"

// CR LF \r\n (windows)
const windowsEOL = "\\r\\n"

// CF \r (mac)
const macEOL = "\\r"

// LF \n (unix)
const unixEOL = "\\n"

// Included at the bottom of each Teams message
var messageTrailer string = fmt.Sprintf("%s %s Message generated by [%s](%s) (%s)",
	breakStatement, breakStatement, myAppName, myAppURL, version)

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

	if webhook.Team == "" {
		return fmt.Errorf("team name too short")

	}

	if webhook.Channel == "" {
		return fmt.Errorf("channel name too short")
	}

	// ensure that at least the prefix + SOMETHING is present; test against
	// the shorter of the two known prefixes
	if len(webhook.WebhookURL) <= len(webhookURLOfficecomPrefix) {
		return fmt.Errorf("incomplete webhook URL: provided URL %q shorter than or equal to just the %q URL prefix",
			webhook.WebhookURL,
			webhookURLOfficecomPrefix,
		)
	}

	// ensure that known/expected prefixes are used
	switch {
	case strings.HasPrefix(webhook.WebhookURL, webhookURLOfficecomPrefix):
	case strings.HasPrefix(webhook.WebhookURL, webhookURLOffice365Prefix):
	default:
		u, err := url.Parse(webhook.WebhookURL)
		if err != nil {
			return fmt.Errorf(
				"unable to parse webhook URL %q: %v",
				webhook.WebhookURL,
				err,
			)
		}
		userProvidedWebhookURLPrefix := u.Scheme + "://" + u.Host

		return fmt.Errorf(
			"webhook URL does not contain expected prefix; got %q, expected one of %q or %q",
			userProvidedWebhookURLPrefix,
			webhookURLOfficecomPrefix,
			webhookURLOffice365Prefix,
		)
	}

	// This is fairly tight validation and will likely require future tending
	matched, err := regexp.MatchString(validWebhookURLRegex, webhook.WebhookURL)
	if !matched {
		return fmt.Errorf(
			"webhook URL does not match expected pattern: %v\n"+
				"got: %q\n"+
				"expected one of:\n"+
				"  * %q\n"+
				"  * %q",
			err,
			webhook.WebhookURL,
			webhookURLOfficecomPrefix+webhookURLOfficialDocsSampleURI,
			webhookURLOffice365Prefix+webhookURLOfficialDocsSampleURI,
		)
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
		return fmt.Errorf("provided message theme color too short; got message %q of length %d, expected length of %d",
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

// convertEOLToBreak converts \r\n (windows), \r (mac) and \n (unix) into <br>
// HTML/Markdown break statements
func convertEOLToBreak(s string) string {

	s = strings.ReplaceAll(s, windowsEOL, breakStatement)
	s = strings.ReplaceAll(s, macEOL, breakStatement)
	s = strings.ReplaceAll(s, unixEOL, breakStatement)

	return s
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

func (tc TeamsChannel) String() string {
	return fmt.Sprintf("Team=%q, Channel=%q, WebhookURL=%q",
		tc.Team,
		tc.Channel,
		tc.WebhookURL,
	)
}

func (tm TeamsMessage) String() string {
	return fmt.Sprintf("ThemeColor=%q, MessageTitle=%q, MessageText=%q",
		tm.ThemeColor,
		tm.MessageTitle,
		tm.MessageText,
	)
}

func main() {

	webhook := TeamsChannel{}
	message := TeamsMessage{}

	var verboseOutput bool
	var silentOutput bool
	var convertEOL bool

	flag.BoolVar(&verboseOutput, "verbose", false, "Whether detailed output should be shown after message submission success or failure")
	flag.BoolVar(&silentOutput, "silent", false, "Whether ANY output should be shown after message submission success or failure")
	flag.BoolVar(&convertEOL, "convert-eol", false, "Whether messages with Windows, Mac and Linux newlines are updated to use break statements before message submission")
	flag.StringVar(&webhook.Team, "team", "", "The name of the Team containing our target channel")
	flag.StringVar(&webhook.Channel, "channel", "", "The target channel where we will send a message")
	flag.StringVar(&webhook.WebhookURL, "url", "", "The Webhook URL provided by a preconfigured Connector")
	flag.StringVar(&message.ThemeColor, "color", defaultMessageThemeColor, "The hex color code used to set the desired trim color on submitted messages")
	flag.StringVar(&message.MessageTitle, "title", "", "The title for the message to submit")
	flag.StringVar(&message.MessageText, "message", "", "The (optionally) Markdown-formatted message to submit")

	// parse flag definitions from the argument list
	flag.Parse()

	// Validate provided info before going any further
	if err := validateMessage(message); err != nil {
		log.Fatal(err)
	}

	if err := validateWebhook(webhook); err != nil {
		log.Fatal(err)
	}

	// TODO: Move elsewhere?
	if silentOutput && verboseOutput {
		fmt.Println("Unsupported: You cannot have both silent and verbose output.")
		os.Exit(1)
	}

	// Convert EOL if user requested it (useful for converting script output)
	if convertEOL {
		message.MessageText = convertEOLToBreak(message.MessageText)
	}

	// setup message card
	msgCard := goteamsnotify.NewMessageCard()
	msgCard.Title = message.MessageTitle
	msgCard.Text = message.MessageText + messageTrailer
	msgCard.ThemeColor = message.ThemeColor

	// FIXME: Work around goteamsnotify package using `log.Println(err)`
	// by directing all statements other than ours to /dev/null
	log.SetOutput(ioutil.Discard)

	if err := sendMessage(webhook.WebhookURL, msgCard); err != nil {

		// Display error output if silence is not requested
		if !silentOutput {
			fmt.Printf("\n\nERROR: Failed to submit message to %q channel in the %q team: %v\n\n",
				webhook.Channel, webhook.Team, err)

			if verboseOutput {
				fmt.Printf("[Message]: %+v\n[Webhook]: %+v\n[Error]: %v", message, webhook, err)
			}

		}

		// Regardless of silent flag, explicitly note unsuccessful results
		os.Exit(1)
	}

	// FIXME: Remove this workaround once the goteamsnotify package is
	// updated or I learn of a better/proper way to handle this
	//
	// By this point any errors emitted by the goteamsnotify package
	// should have already been emitted and then immediately redirected to
	// dev/null, so go ahead and restore logging output
	log.SetOutput(os.Stdout)

	if !silentOutput {

		// Emit basic success message
		log.Println("Message successfully sent!")

		if verboseOutput {
			log.Printf("Webhook: %s\n", webhook)
			log.Printf("Message: %s\n", message)
		}
	}

}
