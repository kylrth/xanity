package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var pollCmd = &cobra.Command{
	Use:   "poll",
	Short: "update arXiv index",
	Long:  `Poll arXiv for more papers, and then compute features for them.`,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		setConfig()

		var err error

		if onlyCompute {
			err = compute()
		} else {
			err = pollAndCompute()
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	},
}

var onlyCompute bool

func init() {
	pollCmd.Flags().BoolVarP(
		&onlyCompute, "only-compute", "c", false, "don't poll, just update features")
}

func pollAndCompute() error {
	gotNew, err := poll()
	if err != nil {
		return fmt.Errorf("poll arXiv: %w", err)
	}

	if gotNew {
		err = compute()
		if err != nil {
			return fmt.Errorf("compute features: %w", err)
		}
	}

	return err
}

// poll runs arxiv_daemon.py and returns true if new papers were added.
func poll() (bool, error) {
	// edit file to add our query string
	pollQuery := viper.GetString("poll.query")
	log.Println("modifying script with new poll query", pollQuery)

	rep := strings.NewReplacer(
		"cat:cs.CV+OR+cat:cs.LG+OR+cat:cs.CL+OR+cat:cs.AI+OR+cat:cs.NE+OR+cat:cs.RO",
		pollQuery,
	)

	data, err := os.ReadFile("arxiv_daemon.py")
	if err != nil {
		return false, err
	}

	newData := rep.Replace(string(data))

	err = os.WriteFile("arxiv_daemon_temp.py", []byte(newData), 0o600)
	if err != nil {
		return false, err
	}

	// run the edited script
	cmd := exec.Command( //nolint:gosec // The customer is always right.
		"python", "-u", "arxiv_daemon_temp.py",
		"--num", strconv.Itoa(viper.GetInt("poll.num")),
		"--start", strconv.Itoa(viper.GetInt("poll.start")),
		"--break-after", strconv.Itoa(viper.GetInt("poll.break")),
	)

	err = run(cmd)
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if exitErr.ExitCode() == 1 {
				log.Println("no new papers found (or polling had an error)")

				return false, nil
			}
		}

		return false, err
	}

	return true, nil
}

func compute() error {
	cmd := exec.Command( //nolint:gosec // The customer is always right.
		"python", "-u", "compute.py",
		"--num", strconv.Itoa(viper.GetInt("compute.features")),
		"--min_df", viper.GetString("compute.min_df"),
		"--max_df", viper.GetString("compute.max_df"),
		"--max_docs", strconv.Itoa(viper.GetInt("compute.max_docs")),
	)

	return run(cmd)
}

// run runs the command and forwards stdout and stderr.
func run(cmd *exec.Cmd) error {
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
