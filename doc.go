// Copyright 2020 Adam Chalkley
//
// https://github.com/atc0005/send2teams
//
// Licensed under the MIT License. See LICENSE file in the project root for
// full license information.

/*

send2teams is a small CLI tool used to submit messages to Microsoft Teams.

PROJECT HOME

See our GitHub repo (https://github.com/atc0005/send2teams) for the latest
code, to file an issue or submit improvements for review and potential
inclusion into the project.

PURPOSE

send2teams is intended for use by Nagios, scripts or other actions that may
need to submit pass/fail results to a MS Teams channel.

FEATURES

• single binary, no outside dependencies

• minimal configuration

• very few build dependencies

• optional conversion of messages with Windows, Mac or Linux newlines to `<br>` to increase compatibility with Teams formatting

USAGE

Help output is below. See the README for examples.

	Usage of T:\github\send2teams\send2teams.exe:
	-channel string
			The target channel where we will send a message
	-color string
			The hex color code used to set the desired trim color on submitted messages (default "#832561")
	-convert-eol
			Whether messages with Windows, Mac and Linux newlines are updated to use break statements before message submission
	-message string
			The (optionally) Markdown-formatted message to submit
	-silent
			Whether ANY output should be shown after message submission success or failure
	-team string
			The name of the Team containing our target channel
	-title string
			The title for the message to submit
	-url string
			The Webhook URL provided by a preconfigured Connector
	-verbose
			Whether detailed output should be shown after message submission success or failure

*/
package main
