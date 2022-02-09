// Copyright 2021 Adam Chalkley
//
// https://github.com/atc0005/send2teams
//
// Licensed under the MIT License. See LICENSE file in the project root for
// full license information.

package config

import (
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	goteamsnotify "github.com/atc0005/go-teams-notify/v2"
	"github.com/atc0005/go-teams-notify/v2/messagecard"
)

const (
	versionFlagHelp                     = "Whether to display application version and then immediately exit application."
	verboseOutputFlagHelp               = "Whether detailed output should be shown after message submission success or failure."
	silentOutputFlagHelp                = "Whether ANY output should be shown after message submission success or failure."
	disableWebhookURLValidationFlagHelp = "Whether webhook URL validation should be disabled. Useful when submitting generated JSON payloads to a service like \"https://httpbin.org/\"."
	ignoreInvalidResponseFlagHelp       = "Whether an invalid response from remote endpoint should be ignored. This is expected if submitting a message to a non-standard webhook URL."
	convertEOLFlagHelp                  = "Whether messages with Windows, Mac and Linux newlines are updated to use break statements before message submission."
	teamNameFlagHelp                    = "The name of the Team containing our target channel. Used in log messages. If not specified, defaults to \"unspecified\"."
	channelNameFlagHelp                 = "The target channel where we will send a message. Used in log messages. If not specified, defaults to \"unspecified\"."
	webhookURLFlagHelp                  = "The Webhook URL provided by a preconfigured Connector."
	targetURLFlagHelp                   = "The target URL and label (specified as comma separated pair) usually visible as a button towards the bottom of the Microsoft Teams message."
	userMentionFlagHelp                 = "The DisplayName and ID of the recipient (specified as comma separated pair) for a user mention."
	themeColorFlagHelp                  = "The hex color code used to set the desired trim color on submitted messages."
	titleFlagHelp                       = "The title for the message to submit."
	messageFlagHelp                     = "The message to submit. This message may be provided in Markdown format."
	senderFlagHelp                      = "The (optional) sending application name or generator of the message this app will attempt to deliver."
	retriesFlagHelp                     = "The number of attempts that this application will make to deliver messages before giving up."
	retriesDelayFlagHelp                = "The number of seconds that this application will wait before making another delivery attempt."
)

// Default flag settings if not overridden by user input
const (
	defaultMessageThemeColor           string = "#832561"
	defaultSilentOutput                bool   = false
	defaultVerboseOutput               bool   = false
	defaultConvertEOL                  bool   = false
	defaultDisableWebhookURLValidation bool   = false
	defaultIgnoreInvalidResponse       bool   = false
	defaultTeamName                    string = "unspecified"
	defaultChannelName                 string = "unspecified"
	defaultWebhookURL                  string = ""
	defaultMessageTitle                string = ""
	defaultMessageText                 string = ""
	defaultSender                      string = ""
	defaultDisplayVersionAndExit       bool   = false
	defaultRetries                     int    = 2
	defaultRetriesDelay                int    = 2
)

// Overridden via Makefile for release builds
var version = "dev build"

// ErrVersionRequested indicates that the user requested application version
// information.
var ErrVersionRequested = errors.New("version information requested")

// Primarily used with branding
const myAppName string = "send2teams"
const myAppURL string = "https://github.com/atc0005/" + myAppName

// teamsSubmissionTimeoutMultiplier is the timeout value for sending messages
// to Microsoft Teams. This value is used along with user specified (or
// default) retries and retries delay values to calculate a context with the
// desired timeout value.
const teamsSubmissionTimeoutMultiplier time.Duration = 2 * time.Second

// DefaultNagiosNotificationTimeout is the default timeout value for Nagios 3
// and 4 installations. This is our *default* timeout ceiling.
const DefaultNagiosNotificationTimeout time.Duration = 30 * time.Second

// brandingTextPrefix is the lead-in or prefix text used to brand or give
// credit to this application for messages delivered to a Microsoft Teams
// channel.
const brandingTextPrefix string = "Message delivered by"

// brandingTextSuffix is the lead-out or suffix text used to give credit to
// the application responsible for generating the text or messages that this
// one will attempt to deliver to a Microsoft Teams channel.
const brandingTextSuffix string = "on behalf of"

// AppInfo identifies common details about the tools provided by this project.
type AppInfo struct {

	// Name specifies the public name shared by all tools in this project.
	Name string

	// Version specifies the public version shared by all tool in this
	// project.
	Version string

	// URL specifies the public repo URL shared by all tools in this project.
	URL string
}

// TargetURL is a URL and description provided by the user for use with
// generating potentialAction entries for display as "buttons" in the
// generated Microsoft Teams message.
type TargetURL struct {

	// URL to be used as the target for labelled "buttons" within a Microsoft
	// Teams message.
	URL url.URL

	// Description is the text used as the label for link "buttons" within a
	// Microsoft Teams message.
	Description string
}

// UserMention is a pair of name and ID values separated by a comma used for
// generating a user mention.
type UserMention struct {
	// ID is the unique identifier for a user that is mentioned. This value
	// can be an object ID (e.g., 5e8b0f4d-2cd4-4e17-9467-b0f6a5c0c4d0) or a
	// UserPrincipalName (e.g., NewUser@contoso.onmicrosoft.com).
	ID string

	// Name is the DisplayName of the user mentioned.
	Name string
}

// Config is a unified set of configuration values for this application. This
// struct is configured via command-line flags provided by the user.
type Config struct {

	// Team is the human-readable name of the Microsoft Teams "team" that
	// contains the channel we wish to post a message to. This is used in
	// informational output produced by this application only; the remote API
	// does not receive this value.
	Team string

	// Channel is human-readable name of the channel within a specific
	// Microsoft Teams "team". This is used in informational output produced
	// by this application only; the remote API does not receive this value.
	Channel string

	// WebhookURL is the full URL used to submit messages to the Teams channel
	// This URL is in the form of https://outlook.office.com/webhook/xxx or
	// https://outlook.office365.com/webhook/xxx. This URL is REQUIRED in
	// order for this application to function and needs to be created in
	// advance by adding/configuring a Webhook Connector in a Microsoft Teams
	// channel that you wish to submit messages to using this application.
	WebhookURL string

	// ThemeColor is a hex color code string representing the desired border
	// trim color for our submitted messages.
	ThemeColor string

	// MessageTitle is the text shown on the top portion of the message "card"
	// that is displayed in Microsoft Teams for the message that we send.
	MessageTitle string

	// MessageText is an (optionally) Markdown-formatted string representing
	// the message that we will submit.
	MessageText string

	// Sender is an optional value provided to indicate what application was
	// responsible for generating the message that this one will attempt to
	// deliver.
	Sender string

	// App represents common details about the tools provided by this project.
	App AppInfo

	// TargetURLs is the collection of user-specified URLs and descriptions
	// that should be displayed as actionable links or "buttons" within the
	// generated Microsoft Teams message.
	TargetURLs targetURLsStringFlag

	// UserMention is the collection of user-specified name and ID values that
	// should be used when generating user mentions within the generated
	// Microsoft Teams message.
	UserMentions userMentionsStringFlag

	// Retries is the number of attempts that this application will make
	// to deliver messages before giving up.
	Retries int

	// RetriesDelay is the number of seconds to wait between retry attempts.
	RetriesDelay int

	// DisableWebhookURLValidation indicates whether validation of the
	// user-specified WebhookURL should be disabled. Useful for testing.
	DisableWebhookURLValidation bool

	// IgnoreInvalidResponse indicates whether an invalid response from remote
	// endpoint should be ignored. This is expected if submitting a message to
	// a non-standard webhook URL.
	IgnoreInvalidResponse bool

	// Whether detailed output should be shown after message submission
	// success or failure.
	VerboseOutput bool

	// Whether ANY output should be shown after message submission success or
	// failure.
	SilentOutput bool

	// Whether messages with Windows, Mac and Linux newlines are updated to
	// use break statements before message submission.
	ConvertEOL bool

	// ShowVersion is a flag indicating whether the user opted to display only
	// the version string and then immediately exit the application
	ShowVersion bool
}

type targetURLsStringFlag []TargetURL

type userMentionsStringFlag []UserMention

// String returns a list of all user-specified target URLs.
func (tus *targetURLsStringFlag) String() string {

	// From the `flag` package docs:
	// "The flag package may call the String method with a zero-valued
	// receiver, such as a nil pointer."
	if tus == nil {
		return ""
	}

	var output strings.Builder

	for i, target := range *tus {
		fmt.Fprintf(
			&output,
			"[URL: %s, Desc: %s]",
			target.URL.String(),
			target.Description,
		)

		// separate the current entry from the next if more to process
		if i+1 != len(*tus) {
			fmt.Fprintf(&output, ", ")
		}
	}

	return output.String()
}

// Set is called once by the flag package, in command line order, for each
// flag present. At most, two comma-separated values are allowed per flag
// invocation in order to specify the target URL and the target URL
// description. An error is returned if more comma-separated values are
// specified than expected or if the provided URL is in an invalid format.
func (tus *targetURLsStringFlag) Set(value string) error {

	// split comma-separated string into multiple values
	items := strings.Split(value, ",")

	// We should only have two items after splitting on the comma, the target
	// URL and its description. Abort if more or less are supplied.
	if len(items) != 2 {
		return fmt.Errorf(
			"received %d arguments for target URL flag, expected 2",
			len(items),
		)
	}

	// prune any leading and trailing whitespace, drop any quotes which might
	// cause issues later.
	for index, item := range items {
		items[index] = strings.TrimSpace(item)
		items[index] = strings.ReplaceAll(items[index], "'", "")
		items[index] = strings.ReplaceAll(items[index], "\"", "")
	}

	u, err := url.Parse(items[0])
	if err != nil {
		return fmt.Errorf(
			"provided URL %s failed to parse: %v",
			items[0],
			err,
		)
	}

	desc := items[1]

	// add them to the collection
	*tus = append(*tus, TargetURL{
		URL:         *u,
		Description: desc,
	})

	return nil
}

// String returns a list of all user-specified user mentions.
func (ums *userMentionsStringFlag) String() string {

	// From the `flag` package docs:
	// "The flag package may call the String method with a zero-valued
	// receiver, such as a nil pointer."
	if ums == nil {
		return ""
	}

	var output strings.Builder

	for i, mention := range *ums {
		fmt.Fprintf(&output, "[Name: %s, ID: %s]", mention.Name, mention.ID)

		// separate the current entry from the next if more to process
		if i+1 != len(*ums) {
			fmt.Fprintf(&output, ", ")
		}
	}

	return output.String()
}

// Set is called once by the flag package, in command line order, for each
// flag present. At most, two comma-separated values are allowed per flag
// invocation in order to specify the name and ID for a user mention. An error
// is returned if more comma-separated values are specified than expected.
func (ums *userMentionsStringFlag) Set(value string) error {

	// split comma-separated string into multiple values
	items := strings.Split(value, ",")

	// Abort unless we have exactly two items after splitting on the comma.
	if len(items) != 2 {
		return fmt.Errorf(
			"received %d arguments for user mention flag, expected 2",
			len(items),
		)
	}

	// prune any leading and trailing whitespace, drop any quotes which might
	// cause issues later.
	for index, item := range items {
		items[index] = strings.TrimSpace(item)
		items[index] = strings.ReplaceAll(items[index], "'", "")
		items[index] = strings.ReplaceAll(items[index], "\"", "")
	}

	// add them to the collection
	*ums = append(*ums, UserMention{
		Name: items[0],
		ID:   items[1],
	})

	return nil
}

// Branding is responsible for emitting application name, version and origin
func Branding() {
	fmt.Fprintf(flag.CommandLine.Output(), "\n%s %s\n%s\n\n", myAppName, version, myAppURL)
}

// MessageTrailer generates a branded "footer" for use with submitted Teams
// messages. If specified, the sending or "generator" application is credited
// as the source of the message, while this application is credited as the
// delivery agent/mechanism.
func MessageTrailer(sender string) string {
	var onBehalfOf string
	if strings.TrimSpace(sender) != "" {
		onBehalfOf = fmt.Sprintf(" %s %s ", brandingTextSuffix, sender)
	}

	return fmt.Sprintf(
		"%s [%s](%s) (%s) at %s%s",
		brandingTextPrefix,
		myAppName,
		myAppURL,
		version,
		time.Now().Format(time.RFC3339),
		onBehalfOf,
	)
}

// flagsUsage displays branding information and general usage details
func flagsUsage() func() {

	return func() {

		myBinaryName := filepath.Base(os.Args[0])

		Branding()

		fmt.Fprintf(flag.CommandLine.Output(), "Usage of \"%s\":\n",
			myBinaryName,
		)
		flag.PrintDefaults()

	}
}

func (c Config) String() string {
	return fmt.Sprintf(
		"Team=%q, "+
			"Channel=%q, "+
			"WebhookURL=%q, "+
			"ThemeColor=%q, "+
			"MessageTitle=%q, "+
			"MessageText=%q, "+
			"Sender=%q, "+
			"TargetURLs=%q, "+
			"Retries=%q, "+
			"RetriesDelay=%q, "+
			"AppTimeout=%q, "+
			"DisableWebhookURLValidation=%t, "+
			"IgnoreInvalidResponse=%t, "+
			"VerboseOutput=%t, "+
			"SilentOutput=%t, "+
			"ConvertEOL=%t",
		c.Team,
		c.Channel,
		c.WebhookURL,
		c.ThemeColor,
		c.MessageTitle,
		c.MessageText,
		c.Sender,
		c.TargetURLs.String(),
		strconv.Itoa(c.Retries),
		strconv.Itoa(c.RetriesDelay),
		c.TeamsSubmissionTimeout(),
		c.DisableWebhookURLValidation,
		c.IgnoreInvalidResponse,
		c.VerboseOutput,
		c.SilentOutput,
		c.ConvertEOL,
	)
}

// NewConfig is a factory function that produces a new Config object based
// on user provided flag values.
func NewConfig() (*Config, error) {
	cfg := Config{}

	cfg.handleFlagsConfig()

	cfg.App = AppInfo{
		Name:    myAppName,
		Version: version,
		URL:     myAppURL,
	}

	// Return immediately if user just wants version details
	if cfg.ShowVersion {
		return &cfg, ErrVersionRequested
	}

	// log.Debug("Validating configuration ...")
	if err := cfg.Validate(cfg.DisableWebhookURLValidation); err != nil {
		flag.Usage()
		return nil, err
	}
	// log.Debug("Configuration validated")

	return &cfg, nil
}

// Validate verifies all struct fields have been provided acceptable values.
func (c Config) Validate(disableWebhookURLValidation bool) error {

	// Current implementation of user mentions is incompatible with most
	// MessageCard settings/values. Future implementation of Adaptive Card
	// support in the atc0005/go-teams-notify library is expected to remove
	// some/most of these incompatibilities.
	switch {
	case c.UserMentions != nil:

		if len(c.TargetURLs) > 0 {
			return fmt.Errorf("target urls flag is incompatible with user mentions flag")
		}

		if c.MessageTitle != "" {
			return fmt.Errorf("message title flag is incompatible with user mentions flag")
		}

		if c.ThemeColor != defaultMessageThemeColor {
			return fmt.Errorf("theme color flag is incompatible with user mentions flag")
		}

	default:
		// Expected pattern: #832561
		if len(c.ThemeColor) < len(defaultMessageThemeColor) {

			expectedLength := len(defaultMessageThemeColor)
			actualLength := len(c.ThemeColor)
			return fmt.Errorf("provided message theme color too short; got message %q of length %d, expected length of %d",
				c.ThemeColor, actualLength, expectedLength)
		}

		if c.MessageTitle == "" {
			return fmt.Errorf("message title too short")
		}

		// We rely on the Set() method for the flag.Value interface to ensure that
		// the required URL and description values are provided for each target
		// URL. We verify here that we don't exceed the maximum supported
		// potentialActions for the `section` that we will generate.
		//
		// https://docs.microsoft.com/en-us/outlook/actionable-messages/message-card-reference#actions
		if len(c.TargetURLs) > messagecard.PotentialActionMaxSupported {
			return fmt.Errorf(
				"%d target URLs specified, a maximum of %d are supported",
				len(c.TargetURLs),
				messagecard.PotentialActionMaxSupported,
			)
		}

	}

	/*
		Shared/common validation checks.
	*/

	if c.SilentOutput && c.VerboseOutput {
		return fmt.Errorf("unsupported: You cannot have both silent and verbose output")
	}

	if c.MessageText == "" {
		return fmt.Errorf("message content too short")
	}

	// Team and Channel names are optional. If provided, use as-is.

	// Sender is optional. If provided, use as-is.

	if c.Retries < 0 {
		return fmt.Errorf("retries too short")
	}

	if c.RetriesDelay < 0 {
		return fmt.Errorf("retries delay too short")
	}

	// Create Microsoft Teams client
	mstClient := goteamsnotify.NewTeamsClient()

	// Allow selective toggling of webhook URL validation.
	if !disableWebhookURLValidation {
		if err := mstClient.ValidateWebhook(c.WebhookURL); err != nil {
			return fmt.Errorf("webhook URL validation failed: %w", err)
		}
	}

	// Indicate that we didn't spot any problems
	return nil

}
