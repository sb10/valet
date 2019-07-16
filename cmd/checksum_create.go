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
 * @file checksum_create.go
 * @author Keith James <kdj@sanger.ac.uk>
 */

package cmd

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/spf13/cobra"
	logf "valet/log/logfacade"
	"valet/valet"
)

var checksumCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create checksum files under a root directory",
	Long: `
valet checksum create will monitor a directory hierarchy and locate data files
within it that have no accompanying checksum file, or have a checksum file that
is stale. valet will then calculate the checksum and create or update the
checksum file.

- Creating up-to-date checksum files
  
  - Directory hierarchy styles supported

    - Any
  
  - File patterns supported

    - *.fast5$
    - *.fastq$

  - Checksum file patterns supported

    - (data file name).md5
`,
	Example: `
valet checksum create --root /data --exclude /data/intermediate \
    --exclude /data/queued_reads --exclude /data/reports \
    --interval 20m --verbose`,
	Run: runChecksumCreateCmd,
}

func init() {
	checksumCreateCmd.Flags().StringVarP(&allCliFlags.rootDir,
		"root", "r", "",
		"the root directory of the monitor")

	err := checksumCreateCmd.MarkFlagRequired("root")
	if err != nil {
		logf.GetLogger().Error().
			Err(err).Msg("failed to mark --root required")
		os.Exit(1)
	}

	checksumCreateCmd.Flags().DurationVarP(&allCliFlags.sweepInterval,
		"interval", "i", defaultSweep,
		"directory sweep interval, minimum 30s")

	checksumCreateCmd.Flags().BoolVar(&allCliFlags.dryRun,
		"dry-run", false,
		"dry-run (make no changes)")

	checksumCreateCmd.Flags().StringArrayVar(&allCliFlags.excludeDirs,
		"exclude", []string{},
		"patterns matching directories to prune "+
			"from both monitoring and interval sweeps")

	checksumCmd.AddCommand(checksumCreateCmd)
}

func runChecksumCreateCmd(cmd *cobra.Command, args []string) {
	log := logf.GetLogger()
	root := allCliFlags.rootDir
	exclude := allCliFlags.excludeDirs
	interval := allCliFlags.sweepInterval
	maxProc := allCliFlags.maxProc
	dryRun := allCliFlags.dryRun

	if interval < minSweep {
		log.Error().Msgf("Invalid interval %s (must be > %s)",
			interval, minSweep)
		os.Exit(1)
	}

	CreateChecksumFiles(root, exclude, interval, maxProc, dryRun)
}

func CreateChecksumFiles(root string, exclude []string, interval time.Duration,
	maxProc int, dryRun bool) {
	log := logf.GetLogger()

	cancelCtx, cancel := context.WithCancel(context.Background())
	setupSignalHandler(cancel)

	pred := valet.RequiresChecksum

	// pruneFn, err := makeRegexPruneFn(exclude)
	pruneFn, err := makeGlobPruneFn(exclude)
	if err != nil {
		log.Error().Err(err).Msg("error in exclusion patterns")
		os.Exit(1)
	}

	wpaths, werrs := valet.WatchFiles(cancelCtx, root, pred, pruneFn)
	fpaths, ferrs := valet.FindFilesInterval(cancelCtx, root, pred, pruneFn, interval)
	mpaths := mergeFileChannels(wpaths, fpaths)
	errs := mergeErrorChannels(werrs, ferrs)

	var workFn valet.WorkFunc
	if dryRun {
		workFn = valet.DoNothing
	} else {
		workFn = valet.CreateOrUpdateMD5ChecksumFile
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		err := valet.ProcessFiles(mpaths, workFn, maxProc)
		if err != nil {
			log.Error().Err(err).Msg("failed processing")
			os.Exit(1)
		}
	}()

	if err := <-errs; err != nil {
		log.Error().Err(err).Msg("failed to complete processing")
		os.Exit(1)
	}

	wg.Wait()
}
