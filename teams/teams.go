package teams

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"regexp"
	"strings"

	//goteamsnotify "gopkg.in/dasrick/go-teams-notify.v1"
	goteamsnotify "github.com/atc0005/go-teams-notify"
)

// logger is a package logger that can be enabled from client code to allow
// logging output from this package when desired/needed for troubleshooting
var logger *log.Logger

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

// TODO: Why is the double leading slash necessary to match on escape
// sequences in order to replace them?
//
// A: Convert double-quoted strings to backtick-quoted strings, replace
// double-backslash with single-backslash as desired.

// Used by Teams to separate lines
const breakStatement = "<br>"

// CR LF \r\n (windows)
const windowsEOLActual = "\r\n"
const windowsEOLEscaped = `\r\n`

// CF \r (mac)
const macEOLActual = "\r"
const macEOLEscaped = `\r`

// LF \n (unix)
const unixEOLActual = "\n"
const unixEOLEscaped = `\n`

// Even though Microsoft Teams doesn't show the additional newlines,
// https://messagecardplayground.azurewebsites.net/ DOES show the results
// as a formatted code block. Including the newlines now is an attempt at
// "future proofing" the codeblock support in MessageCard values sent to
// Microsoft Teams.
const (

	// msTeamsCodeBlockSubmissionPrefix is the prefix appended to text input
	// to indicate that the text should be displayed as a codeblock by
	// Microsoft Teams.
	msTeamsCodeBlockSubmissionPrefix string = "\n```\n"
	// msTeamsCodeBlockSubmissionPrefix string = "```"

	// msTeamsCodeBlockSubmissionSuffix is the suffix appended to text input
	// to indicate that the text should be displayed as a codeblock by
	// Microsoft Teams.
	msTeamsCodeBlockSubmissionSuffix string = "```\n"
	// msTeamsCodeBlockSubmissionSuffix string = "```"

	// msTeamsCodeSnippetSubmissionPrefix is the prefix appended to text input
	// to indicate that the text should be displayed as a code formatted
	// string of text by Microsoft Teams.
	msTeamsCodeSnippetSubmissionPrefix string = "`"

	// msTeamsCodeSnippetSubmissionSuffix is the suffix appended to text input
	// to indicate that the text should be displayed as a code formatted
	// string of text by Microsoft Teams.
	msTeamsCodeSnippetSubmissionSuffix string = "`"
)

func init() {

	// Disable logging output by default unless client code explicitly
	// requests it
	logger = log.New(os.Stderr, "[send2teams/teams] ", 0)
	logger.SetOutput(ioutil.Discard)

}

// EnableLogging enables logging output from this package. Output is muted by
// default unless explicitly requested (by calling this function).
func EnableLogging() {
	logger.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	logger.SetOutput(os.Stderr)
}

// DisableLogging reapplies default package-level logging settings of muting
// all logging output.
func DisableLogging() {
	logger.SetFlags(0)
	logger.SetOutput(ioutil.Discard)
}

// TryToFormatAsCodeBlock acts as a wrapper for FormatAsCodeBlock. If an
// error is encountered in the FormatAsCodeBlock function, this function will
// return the original string, otherwise if no errors occur the newly formatted
// string will be returned.
func TryToFormatAsCodeBlock(input string) string {

	result, err := FormatAsCodeBlock(input)
	if err != nil {
		logger.Printf("TryToFormatAsCodeBlock: error occurred when calling FormatAsCodeBlock: %v\n", err)
		logger.Println("TryToFormatAsCodeBlock: returning original string")
		return input
	}

	logger.Println("TryToFormatAsCodeBlock: no errors occurred when calling FormatAsCodeBlock")
	return result
}

// TryToFormatAsCodeSnippet acts as a wrapper for FormatAsCodeSnippet. If
// an error is encountered in the FormatAsCodeSnippet function, this function will
// return the original string, otherwise if no errors occur the newly formatted
// string will be returned.
func TryToFormatAsCodeSnippet(input string) string {

	result, err := FormatAsCodeSnippet(input)
	if err != nil {
		logger.Printf("TryToFormatAsCodeSnippet: error occurred when calling FormatAsCodeBlock: %v\n", err)
		logger.Println("TryToFormatAsCodeSnippet: returning original string")
		return input
	}

	logger.Println("TryToFormatAsCodeSnippet: no errors occurred when calling FormatAsCodeSnippet")
	return result
}

// FormatAsCodeBlock accepts an arbitrary string, quoted or not, and calls a
// helper function which attempts to format as a valid Markdown code block for
// submission to Microsoft Teams
func FormatAsCodeBlock(input string) (string, error) {

	if input == "" {
		return "", errors.New("received empty string, refusing to format")
	}

	result, err := formatAsCode(
		input,
		msTeamsCodeBlockSubmissionPrefix,
		msTeamsCodeBlockSubmissionSuffix,
	)

	return result, err

}

// FormatAsCodeSnippet accepts an arbitrary string, quoted or not, and calls a
// helper function which attempts to format as a single-line valid Markdown
// code snippet for submission to Microsoft Teams
func FormatAsCodeSnippet(input string) (string, error) {
	if input == "" {
		return "", errors.New("received empty string, refusing to format")
	}

	result, err := formatAsCode(
		input,
		msTeamsCodeSnippetSubmissionPrefix,
		msTeamsCodeSnippetSubmissionSuffix,
	)

	return result, err
}

// formatAsCode is a helper function which accepts an arbitrary string, quoted
// or not, a desired prefix and a suffix for the string and attempts to format
// as a valid Markdown formatted code sample for submission to Microsoft Teams
func formatAsCode(input string, prefix string, suffix string) (string, error) {

	var err error
	var byteSlice []byte

	switch {

	// required; protects against slice out of range panics
	case input == "":
		return "", errors.New("received empty string, refusing to format as code block")

	// If the input string is already valid JSON, don't double-encode and
	// escape the content
	case json.Valid([]byte(input)):
		logger.Printf("DEBUG: input string already valid JSON; input: %+v", input)
		logger.Printf("DEBUG: Calling json.RawMessage([]byte(input)); input: %+v", input)

		// FIXME: Is json.RawMessage() really needed if the input string is *already* JSON?
		// https://golang.org/pkg/encoding/json/#RawMessage seems to imply a different use case.
		byteSlice = json.RawMessage([]byte(input))
		//
		// From light testing, it appears to not be necessary:
		//
		// logger.Printf("DEBUG: Skipping json.RawMessage, converting string directly to byte slice; input: %+v", input)
		// byteSlice = []byte(input)

	default:
		logger.Printf("DEBUG: input string not valid JSON; input: %+v", input)
		logger.Printf("DEBUG: Calling json.Marshal(input); input: %+v", input)
		byteSlice, err = json.Marshal(input)
		if err != nil {
			return "", err
		}
	}

	logger.Println("DEBUG: byteSlice as string:", string(byteSlice))

	var prettyJSON bytes.Buffer

	logger.Println("DEBUG: calling json.Indent")
	err = json.Indent(&prettyJSON, byteSlice, "", "\t")
	if err != nil {
		return "", err
	}
	formattedJSON := prettyJSON.String()

	logger.Println("DEBUG: Formatted JSON:", formattedJSON)

	var codeContentForSubmission string

	// try to prevent "runtime error: slice bounds out of range"
	formattedJSONStartChar := 0
	formattedJSONEndChar := len(formattedJSON) - 1
	if formattedJSONEndChar < 0 {
		formattedJSONEndChar = 0
	}

	// handle cases where the formatted JSON string was not wrapped with
	// double-quotes
	switch {

	// if neither start or end character are double-quotes
	case formattedJSON[formattedJSONStartChar] != '"' && formattedJSON[formattedJSONEndChar] != '"':
		codeContentForSubmission = prefix + string(formattedJSON) + suffix

	// if only start character is not a double-quote
	case formattedJSON[formattedJSONStartChar] != '"':
		logger.Println("[WARN]: escapedFormattedJSON is missing leading double-quote")
		codeContentForSubmission = prefix + string(formattedJSON)

	// if only end character is not a double-quote
	case formattedJSON[formattedJSONEndChar] != '"':
		logger.Println("[WARN]: escapedFormattedJSON is missing trailing double-quote")
		codeContentForSubmission = codeContentForSubmission + suffix

	default:
		codeContentForSubmission = prefix + formattedJSON[1:formattedJSONEndChar] + suffix
	}

	logger.Printf("DEBUG: ... as-is:\n%s\n\n", formattedJSON)

	// this requires that the formattedJSON be at least two characters long
	if len(formattedJSON) > 2 {
		logger.Printf("DEBUG: ... without first and last characters: \n%s\n\n", formattedJSON[formattedJSONStartChar+1:formattedJSONEndChar])
	} else {
		logger.Printf("DEBUG: formattedJSON is less than two chars: \n%s\n\n", formattedJSON)
	}

	logger.Printf("DEBUG: codeContentForSubmission: \n%s\n\n", codeContentForSubmission)

	// err should be nil if everything worked as expected
	return codeContentForSubmission, err

}

// ConvertEOLToBreak converts \r\n (windows), \r (mac) and \n (unix) into <br>
// HTML/Markdown break statements
func ConvertEOLToBreak(s string) string {

	//log.Printf("ConvertEOLToBreak: Received %q", s)

	s = strings.ReplaceAll(s, windowsEOLActual, breakStatement)
	s = strings.ReplaceAll(s, windowsEOLEscaped, breakStatement)
	s = strings.ReplaceAll(s, macEOLActual, breakStatement)
	s = strings.ReplaceAll(s, macEOLEscaped, breakStatement)
	s = strings.ReplaceAll(s, unixEOLActual, breakStatement)
	s = strings.ReplaceAll(s, unixEOLEscaped, breakStatement)

	//log.Printf("ConvertEOLToBreak: Returning %q", s)

	return s
}

// SendMessage is a wrapper function for setting up and using the
// goteamsnotify client to send a message card to Microsoft Teams via a
// webhook URL.
func SendMessage(webhookURL string, message goteamsnotify.MessageCard) error {

	// init the client
	mstClient, err := goteamsnotify.NewClient()
	if err != nil {
		return err
	}

	// attempt to send message, return the pass/fail result to caller
	return mstClient.Send(webhookURL, message)
}

// validateWebhookLength ensures that at least the prefix + SOMETHING is
// present; test against the shorter of the two known prefixes
func validateWebhookLength(webhookURL string) error {

	// FIXME: This is made redundant by the prefix check

	if len(webhookURL) <= len(webhookURLOfficecomPrefix) {
		return fmt.Errorf("incomplete webhook URL: provided URL %q shorter than or equal to just the %q URL prefix",
			webhookURL,
			webhookURLOfficecomPrefix,
		)
	}

	return nil
}

// validateWebhookURLPrefix ensure that known/expected prefixes are used with
// provided webhook URL
func validateWebhookURLPrefix(webhookURL string) error {

	// TODO: Inquire about merging this upstream
	// Reasons:
	//
	// Move urls to constants for easier, less error-prone references
	// User-friendly error messages
	//
	switch {
	case strings.HasPrefix(webhookURL, webhookURLOfficecomPrefix):
	case strings.HasPrefix(webhookURL, webhookURLOffice365Prefix):
	default:
		u, err := url.Parse(webhookURL)
		if err != nil {
			return fmt.Errorf(
				"unable to parse webhook URL %q: %v",
				webhookURL,
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

	return nil
}

// validateWebhookURLRegex applies a regular expression pattern check against
// the provided webhook URL to ensure that the URL matches the expected
// pattern.
func validateWebhookURLRegex(webhookURL string) error {

	// TODO: Consider retiring this validation check due to reliance on fixed
	// pattern (subject to change?)
	// This is fairly tight validation and will likely require future tending
	matched, err := regexp.MatchString(validWebhookURLRegex, webhookURL)
	if !matched {
		return fmt.Errorf(
			"webhook URL does not match expected pattern;\n"+
				"got: %q\n"+
				"expected webhook URL in one of these formats:\n"+
				"  * %q\n"+
				"  * %q\n"+
				"error: %v",
			webhookURL,
			webhookURLOfficecomPrefix+"/"+webhookURLOfficialDocsSampleURI,
			webhookURLOffice365Prefix+"/"+webhookURLOfficialDocsSampleURI,
			err,
		)
	}

	return nil
}

// ValidateWebhook applies validation checks to the specified webhook,
// returning an error for any detected issues.
func ValidateWebhook(webhookURL string) error {

	if err := validateWebhookLength(webhookURL); err != nil {
		return err
	}

	if err := validateWebhookURLPrefix(webhookURL); err != nil {
		return err
	}

	if err := validateWebhookURLRegex(webhookURL); err != nil {
		return err
	}

	// Indicate that we didn't spot any problems
	return nil

}
