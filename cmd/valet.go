/*
 * Copyright (C) 2019. Genome Research Ltd. All rights reserved.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License,
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 * @file valet.go
 * @author Keith James <kdj@sanger.ac.uk>
 */

package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"valet/utilities"
	"valet/valet"

	logf "valet/log/logfacade"
	logz "valet/log/zlog"
)

type cliFlags struct {
	debug         bool          // Enable debug logging
	verbose       bool          // Enable verbose logging
	dryRun        bool          // Enable dry-run mode
	maxProc       int           // The maximum number of threads to use
	sweepInterval time.Duration // The interval at which to perform sweeps
	rootDir       string        // The root directory to monitor
	excludeDirs   []string      // Directories to exclude from monitoring
}

var allCliFlags = &cliFlags{}

var valetCmd = &cobra.Command{
	Use: "valet",
	Long: `
valet is a utility for performing small, but important data management tasks
automatically. Once started, valet will continue working until interrupted by
SIGINT or SIGTERM, when it will stop gracefully.
`,
	Run:     runValetCmd,
	Version: valet.Version,
}

func Execute() {
	if err := valetCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	defaultMaxProc := utilities.Abs(runtime.GOMAXPROCS(runtime.NumCPU()))

	valetCmd.PersistentFlags().BoolVar(&allCliFlags.debug,
		"debug", false,
		"enable debug output")
	valetCmd.PersistentFlags().BoolVar(&allCliFlags.verbose,
		"verbose", false,
		"enable verbose output")
	valetCmd.PersistentFlags().IntVarP(&allCliFlags.maxProc,
		"max-proc", "m", defaultMaxProc,
		"set the maximum number of processes to use")
}

func runValetCmd(cmd *cobra.Command, args []string) {
	if err := cmd.Help(); err != nil {
		logf.GetLogger().Error().Err(err).Msg("help command failed")
		os.Exit(1)
	}
}

func setupLogger(flags *cliFlags) logf.Logger {
	var level logf.Level
	if flags.debug {
		level = logf.DebugLevel
	} else if flags.verbose {
		level = logf.InfoLevel
	} else {
		level = logf.ErrorLevel
	}

	// Choose a Zerolog logging backend
	var writer io.Writer
	if terminal.IsTerminal(int(os.Stdout.Fd())) {
		writer = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	} else {
		writer = os.Stderr
	}

	// Synchronize writes to the global logger
	zlogger := logz.New(zerolog.SyncWriter(writer), level)

	return logf.InstallLogger(zlogger)
}

func setupSignalHandler(cancel context.CancelFunc) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		s := <-signals
		log := logf.GetLogger()

		switch s {
		case syscall.SIGINT:
			log.Info().Msg("got SIGINT, shutting down")
			cancel()
		case syscall.SIGTERM:
			log.Info().Msg("got SIGTERM, shutting down")
			cancel()
		default:
			log.Error().Str("signal", s.String()).
				Msg("got unexpected signal, exiting")
			os.Exit(1)
		}
	}()
}
