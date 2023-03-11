package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"

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
	cmd := exec.Command( //nolint:gosec // The customer is always right.
		"python", "arxiv_daemon.py",
		"--num", strconv.Itoa(viper.GetInt("poll.num")),
		"--start", strconv.Itoa(viper.GetInt("poll.start")),
		"--break-after", strconv.Itoa(viper.GetInt("poll.break")),
	)

	err := run(cmd)
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if exitErr.ExitCode() == 1 {
				log.Println("no new papers found, or polling had an error")

				return false, nil
			}
		}

		return false, err
	}

	return true, nil
}

func compute() error {
	cmd := exec.Command(
		"python", "compute.py",
	)

	return run(cmd)
}

// run runs the command and forwards stdout and stderr.
func run(cmd *exec.Cmd) error {
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
