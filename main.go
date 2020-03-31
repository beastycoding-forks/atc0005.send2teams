// Copyright 2019 Adam Chalkley
//
// https://github.com/atc0005/send2teams
//
// Licensed under the MIT License. See LICENSE file in the project root for
// full license information.

package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"

	//goteamsnotify "gopkg.in/dasrick/go-teams-notify.v1"

	// temporarily use our fork while developing changes for potential
	// inclusion in the upstream project
	goteamsnotify "github.com/atc0005/go-teams-notify"
	"github.com/atc0005/send2teams/config"
	"github.com/atc0005/send2teams/teams"
)

func main() {

	// Toggle library debug logging output
	goteamsnotify.EnableLogging()
	// goteamsnotify.DisableLogging()

	//log.Debug("Initializing application")

	cfg, err := config.NewConfig()
	switch {
	// TODO: How else to guard against nil cfg object?
	case cfg != nil && cfg.ShowVersion:
		config.Branding()
		os.Exit(0)
	case err == nil:
		// do nothing for this one
	case errors.Is(err, flag.ErrHelp):
		os.Exit(0)
	default:
		fmt.Printf("failed to initialize application: %s", err)
		os.Exit(1)
	}

	if cfg.VerboseOutput {
		log.Printf("Configuration: %s\n", cfg)
	}

	// Convert EOL if user requested it (useful for converting script output)
	if cfg.ConvertEOL {
		cfg.MessageText = teams.ConvertEOLToBreak(cfg.MessageText)
	}

	// setup message card
	msgCard := goteamsnotify.NewMessageCard()
	msgCard.Title = cfg.MessageTitle
	msgCard.Text = "placeholder (top-level text content)"
	msgCard.ThemeColor = cfg.ThemeColor

	mainMsgSection := goteamsnotify.NewMessageCardSection()

	// This represents what the user would provide via CLI flag:
	mainMsgSection.Text = cfg.MessageText + " (section text)"

	//log.Printf("msgCard before adding mainMsgSection: %+v", msgCard)
	msgCard.AddSection(mainMsgSection)
	//log.Printf("msgCard after adding mainMsgSection: %+v", msgCard)

	/*

		Code Snippet Sample Section

	*/

	codeSnippetSampleSection := goteamsnotify.NewMessageCardSection()
	codeSnippetSampleSection.StartGroup = true

	codeSnippetSampleSection.Title = "Code Snippet Sample Section"

	// This represents something programatically generated:
	unformattedTextSample := "GET request received on /api/v1/echo/json endpoint"
	formattedTextSample, err := goteamsnotify.FormatAsCodeSnippet(unformattedTextSample)
	if err != nil {

		log.Printf("error formatting text as code snippet: %#v", err)
		log.Printf("Current state of section: %+v", codeSnippetSampleSection)

		log.Println("Using unformattedTextSample")
		codeSnippetSampleSection.Text = unformattedTextSample
	} else {
		log.Println("Using formattedTextSample")
		codeSnippetSampleSection.Text = formattedTextSample
		msgCard.AddSection(codeSnippetSampleSection)
	}

	/*

		Code Block Sample Section

	*/

	codeBlockSampleSection := goteamsnotify.NewMessageCardSection()
	codeBlockSampleSection.Title = "Code Block Sample Section"

	// This represents something programatically generated:
	sampleJSONInput := `{"result":{"sourcetype":"mongod","count":"8"},"sid":"scheduler_admin_search_W2_at_14232356_132","results_link":"http://web.example.local:8000/app/search/@go?sid=scheduler_admin_search_W2_at_14232356_132","search_name":null,"owner":"admin","app":"search"}`
	formattedTextSample, err = goteamsnotify.FormatAsCodeBlock(sampleJSONInput)
	if err != nil {

		log.Printf("error formatting text as code snippet: %#v", err)
		log.Printf("Current state of section: %+v", codeBlockSampleSection)

		log.Println("Using unformattedTextSample")
		codeBlockSampleSection.Text = unformattedTextSample
	} else {
		log.Println("Using formattedTextSample")
		codeBlockSampleSection.Text = formattedTextSample
	}

	msgCard.AddSection(codeBlockSampleSection)

	// Setup branding
	trailerSection := goteamsnotify.NewMessageCardSection()
	trailerSection.Text = config.MessageTrailer()
	trailerSection.StartGroup = true

	//log.Printf("msgCard before adding trailerSection: %+v", msgCard)
	msgCard.AddSection(trailerSection)
	//log.Printf("msgCard after adding trailerSection: %+v", msgCard)

	if err := teams.SendMessage(cfg.WebhookURL, msgCard); err != nil {

		// Display error output if silence is not requested
		if !cfg.SilentOutput {
			fmt.Printf("\n\nERROR: Failed to submit message to %q channel in the %q team: %v\n\n",
				cfg.Channel, cfg.Team, err)

			if cfg.VerboseOutput {
				fmt.Printf("[Config]: %+v\n[Error]: %v", cfg, err)
			}

		}

		// Regardless of silent flag, explicitly note unsuccessful results
		os.Exit(1)
	}

	if !cfg.SilentOutput {

		// Emit basic success message
		log.Println("Message successfully sent!")

	}

}
