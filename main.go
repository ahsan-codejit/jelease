package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	jira "github.com/andygrunwald/go-jira"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

var (
	jiraClient *jira.Client
	config     Config
	logger     *log.Logger
)

// Config contains configuration values from environment and .env file.
// Environment takes precedence over the .env file in case of conflicts.
type Config struct {
	Port          string   `envconfig:"PORT" default:"8080"`
	JiraUrl       string   `envconfig:"JIRA_URL" required:"true"`
	JiraUser      string   `envconfig:"JIRA_USER" required:"true"`
	JiraToken     string   `envconfig:"JIRA_TOKEN" required:"true"`
	Project       string   `envconfig:"PROJECT" required:"true"`
	DefaultStatus string   `envconfig:"DEFAULT_STATUS" required:"true"`
	AddLabels     []string `envconfig:"ADD_LABELS"`
}

// Release object unmarshaled from the newreleases.io webhook.
// Some fields omitted for simplicity, refer to the documentation at https://newreleases.io/webhooks
type Release struct {
	Provider string `json:"provider"`
	Project  string `json:"project"`
	Version  string `json:"version"`
}

// Generates a Textual summary for the release, intended to be used as the Jira issue summary
func (release Release) IssueSummary() string {
	return fmt.Sprintf("Update %v to version %v", release.Project, release.Version)
}

func (release Release) JiraIssue() jira.Issue {
	labels := append(config.AddLabels, release.Project)
	return jira.Issue{
		Fields: &jira.IssueFields{
			Description: "Update issue generated by https://github.2rioffice.com/platform/jelease using newreleases.io.",
			Project: jira.Project{
				Key: config.Project,
			},
			Type: jira.IssueType{
				Name: "Task",
			},
			Status: &jira.Status{
				Name: config.DefaultStatus,
			},
			Labels:  labels,
			Summary: release.IssueSummary(),
		},
	}
}

// handleGetRoot handles to GET requests for a basic reachability check
func handleGetRoot(w http.ResponseWriter, r *http.Request) {
	logger.Println("Received health check request")
	io.WriteString(w, "Ok")
}

// handlePostWebhook handles newreleases.io webhook post requests
func handlePostWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		logger.Printf("Rejected request because: %v %v. Attempted method: %v",
			http.StatusMethodNotAllowed, http.StatusText(http.StatusMethodNotAllowed), r.Method)
		return
	}
	// parse newreleases.io webhook
	decoder := json.NewDecoder(r.Body)
	var release Release
	err := decoder.Decode(&release)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		logger.Printf("Couldn't decode request body to json: %v\n error: %v\n", r.Body, err)
		return
	}

	// look for existing update tickets
	existingIssuesQuery := fmt.Sprintf("status = %q and labels = %q", config.DefaultStatus, release.Project)
	existingIssues, resp, err := jiraClient.Issue.Search(existingIssuesQuery, &jira.SearchOptions{})
	if err != nil {
		body, readErr := io.ReadAll(resp.Body)
		errCtx := errors.New("error response from Jira when searching previous issues")
		if readErr != nil {
			logger.Printf("%v: %v. Failed to decode response body: %v", errCtx, err, string(body))
		} else {
			logger.Printf("%v: %v. Response body: %v", errCtx, err, string(body))
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if len(existingIssues) == 0 {
		// no previous issues, create new jira issue
		i := release.JiraIssue()
		newIssue, response, err := jiraClient.Issue.Create(&i)
		if err != nil {
			body, readErr := io.ReadAll(response.Body)
			errCtx := errors.New("error response from Jira when creating issue")
			if readErr != nil {
				logger.Printf("%v: %v. Failed to decode response body: %v", errCtx, err, readErr)
			} else {
				logger.Printf("%v: %v. Response body: %v", errCtx, err, string(body))
			}
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		logger.Printf("Created issue %v\n", newIssue.ID)
		return
	}

	// in case of duplicate issues, update the oldest (probably original) one, ignore rest as duplicates
	var oldestExistingIssue jira.Issue
	var duplicateIssueKeys []string
	for i, existingIssue := range existingIssues {
		if i == 0 {
			oldestExistingIssue = existingIssue
			continue
		}
		tCurrent := time.Time(existingIssue.Fields.Created)
		tOldest := time.Time(oldestExistingIssue.Fields.Created)
		if tCurrent.Before(tOldest) {
			duplicateIssueKeys = append(duplicateIssueKeys, oldestExistingIssue.Key)
			oldestExistingIssue = existingIssue
		} else {
			duplicateIssueKeys = append(duplicateIssueKeys, existingIssue.Key)
		}
	}
	if len(duplicateIssueKeys) > 0 {
		logger.Printf("Ignoring the following possible duplicate issues in favor of older issue %v: %v", oldestExistingIssue.Key,
			strings.Join(duplicateIssueKeys, ", "))
	}

	// This seems hacky, but is taken from the official examples
	// https://github.com/andygrunwald/go-jira/blob/47d27a76e84da43f6e27e1cd0f930e6763dc79d7/examples/addlabel/main.go
	// There is also a jiraClient.Issue.Update() method, but it panics and does not provide a usage example
	type summaryUpdate struct {
		Set string `json:"set" structs:"set"`
	}
	type issueUpdate struct {
		Summary []summaryUpdate `json:"summary" structs:"summary"`
	}
	previousSummary := oldestExistingIssue.Fields.Summary
	updates := map[string]any{
		"update": issueUpdate{
			Summary: []summaryUpdate{
				{Set: release.IssueSummary()},
			},
		},
	}
	resp, err = jiraClient.Issue.UpdateIssue(oldestExistingIssue.ID, updates)
	if err != nil {
		body, readErr := io.ReadAll(resp.Body)
		errCtx := errors.New("error response from Jira when updating issue")
		if readErr != nil {
			logger.Printf("%v: %v. Failed to decode response body: %v", errCtx, err, readErr)
		} else {
			logger.Printf("%v: %v. Response body: %v", errCtx, err, body)
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	logger.Printf("Updated issue summary from %q to %q", previousSummary, release.IssueSummary())
}

func init() {
	logger = log.New(os.Stderr, "", log.LstdFlags|log.Lshortfile)
}

func main() {
	err := run()
	if errors.Is(err, http.ErrServerClosed) {
		logger.Println("server closed")
	} else if err != nil {
		logger.Println(err.Error())
		os.Exit(1)
	}
}

func run() error {
	configSetup := func() error {
		err := godotenv.Load()
		if err != nil {
			logger.Println("No .env file found.")
		}

		err = envconfig.Process("jelease", &config)
		if err != nil {
			return err
		}

		logger.Printf("Jira URL: %v\n", config.JiraUrl)
		tp := jira.BasicAuthTransport{
			Username: config.JiraUser,
			Password: config.JiraToken,
		}
		jiraClient, err = jira.NewClient(tp.Client(), config.JiraUrl)
		if err != nil {
			return fmt.Errorf("failed to create jira client: %w", err)
		}
		return nil
	}

	projectExists := func() error {
		allProjects, response, err := jiraClient.Project.GetList()
		if err != nil {
			body, readErr := io.ReadAll(response.Body)
			errCtx := errors.New("error response from Jira when retrieving project list")
			if readErr != nil {
				return fmt.Errorf("%v: %w. Failed to decode response body: %v", errCtx, err, readErr)
			}
			return fmt.Errorf("%v: %w. Response body: %v", errCtx, err, string(body))
		}
		var projectExists bool
		for _, project := range *allProjects {
			if project.Key == config.Project {
				projectExists = true
				break
			}
		}
		if !projectExists {
			return fmt.Errorf("project %v does not exist on your Jira server", config.Project)
		}
		return nil
	}

	statusExists := func() error {
		allStatuses, response, err := jiraClient.Status.GetAllStatuses()
		if err != nil {
			body, readErr := io.ReadAll(response.Body)
			errCtx := errors.New("error response from Jira when retrieving status list: %+v")
			if readErr != nil {
				return fmt.Errorf("%v: %w. Failed to decode response body: %v", errCtx, err, readErr)
			}
			return fmt.Errorf("%v: %w. Response body: %v", errCtx, err, string(body))
		}
		var statusExists bool
		for _, status := range allStatuses {
			if status.Name == config.DefaultStatus {
				statusExists = true
				break
			}
		}
		if !statusExists {
			return fmt.Errorf("status %v does not exist on your Jira server", config.DefaultStatus)
		}
		return nil
	}

	serveHTTP := func() error {
		http.HandleFunc("/webhook", handlePostWebhook)
		http.HandleFunc("/", handleGetRoot)
		logger.Printf("Listening on port %v\n", config.Port)
		return http.ListenAndServe(fmt.Sprintf(":%v", config.Port), nil)
	}

	err := configSetup()
	if err != nil {
		return fmt.Errorf("error in config setup: %w", err)
	}
	err = projectExists()
	if err != nil {
		return fmt.Errorf("error in check if configured project exists: %w", err)
	}
	err = statusExists()
	if err != nil {
		return fmt.Errorf("error in check if configured default status exists: %w", err)
	}
	return serveHTTP()
}
