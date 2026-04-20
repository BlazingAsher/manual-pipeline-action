package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// stringSlice is a repeatable flag (--success A --success B).
type stringSlice []string

func (s *stringSlice) String() string  { return strings.Join(*s, ", ") }
func (s *stringSlice) Set(v string) error { *s = append(*s, v); return nil }

func main() {
	os.Exit(run())
}

func run() int {
	var (
		question        string
		serverURL       string
		token           string
		timeout         time.Duration
		timeoutExitCode int
		successAnswers  stringSlice
		failureAnswers  stringSlice
	)

	flag.StringVar(&question, "question", "", "Question to ask (required)")
	flag.StringVar(&serverURL, "server", "", "Server base URL, e.g. http://localhost:8080 (required)")
	flag.StringVar(&token, "token", "", "Auth token for the server (required)")
	flag.DurationVar(&timeout, "timeout", 0, "How long to wait for a response, e.g. 10m (required)")
	flag.IntVar(&timeoutExitCode, "timeout-exit-code", 0, "Exit code when the timeout is reached (0 or 1)")
	flag.Var(&successAnswers, "success", "Answer that causes exit 0 (repeatable)")
	flag.Var(&failureAnswers, "failure", "Answer that causes exit 1 (repeatable)")
	flag.Parse()

	if question == "" {
		fmt.Fprintln(os.Stderr, "error: --question is required")
		flag.Usage()
		return 2
	}
	if serverURL == "" {
		fmt.Fprintln(os.Stderr, "error: --server is required")
		flag.Usage()
		return 2
	}
	if token == "" {
		fmt.Fprintln(os.Stderr, "error: --token is required")
		flag.Usage()
		return 2
	}
	if timeout <= 0 {
		fmt.Fprintln(os.Stderr, "error: --timeout is required and must be positive")
		flag.Usage()
		return 2
	}
	if len(successAnswers)+len(failureAnswers) == 0 {
		fmt.Fprintln(os.Stderr, "error: at least one --success or --failure answer is required")
		flag.Usage()
		return 2
	}
	if timeoutExitCode != 0 && timeoutExitCode != 1 {
		fmt.Fprintln(os.Stderr, "error: --timeout-exit-code must be 0 or 1")
		return 2
	}

	serverURL = strings.TrimRight(serverURL, "/")

	// Build combined options list: success answers first, then failure answers.
	allOptions := append([]string{}, successAnswers...)
	allOptions = append(allOptions, failureAnswers...)

	// Build exit-code lookup.
	exitCodes := make(map[string]int, len(allOptions))
	for _, a := range successAnswers {
		exitCodes[a] = 0
	}
	for _, a := range failureAnswers {
		exitCodes[a] = 1
	}

	// Context: cancelled by timeout or Ctrl+C.
	ctx, cancelTimeout := context.WithTimeout(context.Background(), timeout)
	defer cancelTimeout()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	// Create the question on the server.
	result, err := createQuestion(ctx, serverURL, token, question, allOptions)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating question: %v\n", err)
		return 2
	}
	fmt.Fprintf(os.Stderr, "question: %s\n", question)
	fmt.Fprintf(os.Stderr, "options:  %s\n", strings.Join(allOptions, " | "))
	fmt.Fprintf(os.Stderr, "respond:  %s\n", result.interactionURL)

	// Poll until answered, timed out, or interrupted.
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-sigCh:
			return 1

		case <-ctx.Done():
			return timeoutExitCode

		case <-ticker.C:
			answer, err := poll(ctx, result.pollURL)
			if err != nil {
				// Transient error — keep trying until timeout.
				fmt.Fprintf(os.Stderr, "poll error: %v\n", err)
				continue
			}
			if answer == nil {
				continue
			}

			fmt.Println(*answer)

			code, known := exitCodes[*answer]
			if !known {
				// Answer not in either list — treat as failure.
				return 1
			}
			return code
		}
	}
}

type questionResult struct {
	interactionURL string
	pollURL        string
}

func createQuestion(ctx context.Context, serverURL, token, question string, options []string) (questionResult, error) {
	body, _ := json.Marshal(map[string]any{
		"question": question,
		"options":  options,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		serverURL+"/questions", bytes.NewReader(body))
	if err != nil {
		return questionResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return questionResult{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return questionResult{}, fmt.Errorf("server returned %s", resp.Status)
	}

	var result struct {
		InteractionURL string `json:"interaction_url"`
		PollURL        string `json:"poll_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return questionResult{}, fmt.Errorf("decode response: %w", err)
	}
	return questionResult{interactionURL: result.InteractionURL, pollURL: result.PollURL}, nil
}

func poll(ctx context.Context, pollURL string) (*string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pollURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned %s", resp.Status)
	}

	var result struct {
		Answer *string `json:"answer"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return result.Answer, nil
}
