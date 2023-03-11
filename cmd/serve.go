package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/robfig/cron/v3"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "serve arxiv-sanity-lite",
	Long:  `Run arxiv-sanity-lite along with all of its supporting scheduled operations.`,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		setConfig()

		if err := serve(); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	},
}

func serve() error {
	// if /data doesn't exist or is empty, create it and then poll for the first time
	fromScratch, err := noData("/data")
	if err != nil {
		return fmt.Errorf("check database: %w", err)
	}

	log.Println("starting database from scratch:", fromScratch)

	if fromScratch {
		// create the directory if it doesn't exist
		err = os.MkdirAll("/data", 0o777)
		if err != nil {
			return err
		}

		// poll for the first time
		err = pollAndCompute()
		if err != nil {
			return err
		}
	}

	// schedule periodic tasks
	log.Println("scheduling tasks")

	scheduler := cron.New()

	err = schedule(scheduler)
	if err != nil {
		return err
	}

	scheduler.Start()

	// catch shutdown signals
	exitsignal := make(chan os.Signal, 1)
	signal.Notify(exitsignal, syscall.SIGINT, syscall.SIGTERM)

	// start server
	go func() {
		cmd := exec.Command(
			"flask", "run",
			"--port=80", "--host=0.0.0.0", // necessary for docker
		)
		cmd.Env = []string{"FLASK_APP=serve.py"}

		log.Println("starting web server")

		err := run(cmd)
		if err != nil {
			fmt.Printf("error from web server: %v\n", err)
			exitsignal <- syscall.SIGINT
		}
	}()

	// wait for signal or server error
	<-exitsignal

	// wait for any running jobs to finish
	context := scheduler.Stop()
	<-context.Done()

	return nil
}

// noData returns true if the data directory doesn't exist or if it's empty.
func noData(path string) (bool, error) {
	dir, err := isDir(path)
	if err != nil || !dir {
		return true, err
	}

	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1)
	if errors.Is(err, io.EOF) {
		return true, nil
	} else if errors.Is(err, os.ErrNotExist) {
		return true, nil
	}

	return false, err
}

// isDir returns true if the path exists and is a directory.
func isDir(path string) (bool, error) {
	fi, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, err
	}

	return fi.IsDir(), nil
}

func schedule(scheduler *cron.Cron) error {
	// schedule arXiv polling
	if viper.GetBool("poll.enabled") {
		_, err := scheduler.AddFunc(viper.GetString("poll.cron"), func() {
			log.Println("polling arXiv")
			err := pollAndCompute()
			if err != nil {
				log.Printf("error while polling: %v\n", err)
			}
		})
		if err != nil {
			return fmt.Errorf("invalid polling cron string: %w", err)
		}
	}

	// schedule mail
	if viper.GetBool("mail.enabled") {
		log.Println("FATAL: mail is not implemented yet, but it is enabled")

		_, err := scheduler.AddFunc(viper.GetString("mail.cron"), func() {
			log.Println("sending mail")
			log.Println("sent absolutely ZERO mail because it's not implemented!")
		})
		if err != nil {
			return fmt.Errorf("invalid mail cron string: %w", err)
		}
	}

	return nil
}
