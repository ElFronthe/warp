/*
 * Warp (C) 2019-2020 MinIO, Inc.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package cli

import (
	"github.com/minio/cli"
	"github.com/minio/pkg/v3/console"
	"github.com/minio/warp/pkg/bench"
)

var statBucketFlags = []cli.Flag{}

var StatBucketCombinedFlags = combineFlags(globalFlags, ioFlags, statBucketFlags, benchFlags, analyzeFlags)

var statBucketCmd = cli.Command{
	Name:   "stat-bucket",
	Usage:  "benchmark stat bucket (head bucket)",
	Action: mainStatBucket,
	Before: setGlobalsFromContext,
	Flags:  StatBucketCombinedFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS]

DESCRIPTION:
  Benchmark the HeadBucket (StatBucket) operation by repeatedly checking if
  the target bucket exists. Useful for measuring metadata-only request latency
  and throughput without any object data transfer.

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}`,
}

func mainStatBucket(ctx *cli.Context) error {
	checkStatBucketSyntax(ctx)

	b := bench.StatBucket{
		Common: getCommon(ctx, nil),
	}
	return runBench(ctx, &b)
}

func checkStatBucketSyntax(ctx *cli.Context) {
	if ctx.NArg() > 0 {
		console.Fatal("Command takes no arguments")
	}
	checkAnalyze(ctx)
	checkBenchmark(ctx)
}
