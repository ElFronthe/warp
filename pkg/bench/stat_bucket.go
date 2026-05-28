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

package bench

import (
	"context"
	"sync"
	"time"
)

// StatBucket benchmarks HeadBucket speed.
type StatBucket struct {
	Common
}

// Prepare ensures the bucket exists.
func (g *StatBucket) Prepare(ctx context.Context) error {
	cl, done := g.Client()
	defer done()

	exists, err := cl.BucketExists(ctx, g.Bucket)
	if err != nil {
		return err
	}
	if !exists {
		return g.createEmptyBucket(ctx)
	}
	return nil
}

// Start will execute the main benchmark.
func (g *StatBucket) Start(ctx context.Context, wait chan struct{}) error {
	var wg sync.WaitGroup
	wg.Add(g.Concurrency)
	c := g.Collector
	if g.AutoTermDur > 0 {
		ctx = c.AutoTerm(ctx, "STAT_BUCKET", g.AutoTermScale, autoTermCheck, autoTermSamples, g.AutoTermDur)
	}
	// Non-terminating context.
	nonTerm := context.Background()

	for i := 0; i < g.Concurrency; i++ {
		go func(i int) {
			rcv := c.Receiver()
			defer wg.Done()
			done := ctx.Done()

			<-wait
			for {
				select {
				case <-done:
					return
				default:
				}

				if g.rpsLimit(ctx) != nil {
					return
				}

				client, cldone := g.Client()
				op := Operation{
					OpType:   "STAT-BUCKET",
					Thread:   uint32(i),
					Size:     0,
					File:     g.Bucket,
					ObjPerOp: 1,
					Endpoint: client.EndpointURL().String(),
				}

				op.Start = time.Now()
				exists, err := client.BucketExists(nonTerm, g.Bucket)
				op.End = time.Now()

				if err != nil {
					g.Error("BucketExists error: ", err)
					op.Err = err.Error()
					rcv <- op
					cldone()
					continue
				}
				if !exists {
					op.Err = "bucket does not exist"
					g.Error(op.Err)
				}
				rcv <- op
				cldone()
			}
		}(i)
	}
	wg.Wait()
	return nil
}

// Cleanup is a no-op since we don't create test objects.
func (g *StatBucket) Cleanup(ctx context.Context) {
}
