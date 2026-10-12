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

package generator

import (
	"bytes"
	"io"
	"math/rand"
	"testing"
)

func TestFixedContentIdentical(t *testing.T) {
	srcFn, err := NewFn(WithRandomData().Apply(), WithSize(64<<10), WithFixedContent(true))
	if err != nil {
		t.Fatalf("NewFn() error = %v", err)
	}
	a, b := srcFn(), srcFn()

	var want []byte
	for i := 0; i < 10; i++ {
		obj := a.Object()
		got, err := io.ReadAll(obj.Reader)
		if err != nil {
			t.Fatalf("Object(%d) ReadAll error = %v", i, err)
		}
		if len(got) != 64<<10 {
			t.Fatalf("Object(%d) size = %d, want %d", i, len(got), 64<<10)
		}
		if want == nil {
			want = got
			continue
		}
		if !bytes.Equal(want, got) {
			t.Fatalf("Object(%d) content differs from the first object", i)
		}
	}

	// A second source, as handed to another worker, must stream the same bytes.
	obj := b.Object()
	got, err := io.ReadAll(obj.Reader)
	if err != nil {
		t.Fatalf("second source ReadAll error = %v", err)
	}
	if !bytes.Equal(want, got) {
		t.Fatal("second source content differs from the first source")
	}
}

func TestFixedContentUniqueNames(t *testing.T) {
	src, err := New(WithRandomData().Apply(), WithSize(1024), WithFixedContent(true))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	seen := make(map[string]bool, 20)
	for i := 0; i < 20; i++ {
		obj := src.Object()
		if seen[obj.Name] {
			t.Errorf("Object(%d) name %q duplicates a previous object", i, obj.Name)
		}
		seen[obj.Name] = true
	}
}

func TestFixedContentRejectsVaryingSizes(t *testing.T) {
	tests := []struct {
		name string
		opts []Option
	}{
		{
			name: "random size",
			opts: []Option{WithRandomData().Apply(), WithSize(64 << 10), WithRandomSize(true), WithFixedContent(true)},
		},
		{
			name: "size histogram",
			opts: []Option{WithRandomData().Apply(), WithSizeHistograms("1024:1,2048:1"), WithFixedContent(true)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := New(tt.opts...); err == nil {
				t.Fatal("expected error combining fixed content with varying sizes")
			}
		})
	}
}

func TestEachObjectUnique(t *testing.T) {
	src, err := New(WithSize(64<<10), WithRandomData().Apply())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	seen := make(map[string]bool, 20)
	for i := 0; i < 20; i++ {
		obj := src.Object()
		b, err := io.ReadAll(obj.Reader)
		if err != nil {
			t.Fatalf("Object(%d) ReadAll error: %v", i, err)
		}
		if len(b) != 64<<10 {
			t.Fatalf("Object(%d) size = %d, want %d", i, len(b), 64<<10)
		}
		h := string(b[:64])
		if seen[h] {
			t.Errorf("Object(%d) content duplicates a previous object", i)
		}
		seen[h] = true
	}
}

func TestEachObjectUniqueRandomSize(t *testing.T) {
	src, err := New(WithMinMaxSize(128, 64<<10), WithRandomSize(true), WithRandomData().Apply())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	seen := make(map[string]bool, 20)
	for i := 0; i < 20; i++ {
		obj := src.Object()
		b, err := io.ReadAll(obj.Reader)
		if err != nil {
			t.Fatalf("Object(%d) ReadAll error: %v", i, err)
		}
		if len(b) < 128 || len(b) > 64<<10 {
			t.Fatalf("Object(%d) size = %d out of range [%d, %d]", i, len(b), 128, 64<<10)
		}
		h := string(b)
		if seen[h] {
			t.Errorf("Object(%d) content duplicates a previous object", i)
		}
		seen[h] = true
	}
}

func TestEachSmallObjectUnique(t *testing.T) {
	src, err := New(WithSize(256), WithRandomData().Apply())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	seen := make(map[string]bool, 20)
	for i := 0; i < 20; i++ {
		obj := src.Object()
		b, err := io.ReadAll(obj.Reader)
		if err != nil {
			t.Fatalf("Object(%d) ReadAll error: %v", i, err)
		}
		if len(b) != 256 {
			t.Fatalf("Object(%d) size = %d, want 256", i, len(b))
		}
		h := string(b)
		if seen[h] {
			t.Errorf("Object(%d) content duplicates a previous object", i)
		}
		seen[h] = true
	}
}

func TestNew(t *testing.T) {
	type args struct {
		opts []Option
	}
	tests := []struct {
		name     string
		args     args
		wantErr  bool
		wantSize int
	}{
		{
			name: "Default",
			args: args{
				opts: nil,
			},
			wantErr:  false,
			wantSize: 1 << 20,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.args.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				t.Error(err)
				return
			}
			if got == nil {
				t.Errorf("New() got = nil, want not nil")
				return
			}
			obj := got.Object()
			b, err := io.ReadAll(obj.Reader)
			if err != nil {
				t.Error(err)
				return
			}
			if len(b) != tt.wantSize {
				t.Errorf("New() size = %v, wantSize = %v", len(b), tt.wantSize)
				return
			}
			n, err := obj.Reader.Seek(0, 0)
			if err != nil {
				t.Error(err)
				return
			}
			if n != 0 {
				t.Errorf("Expected 0, got %v", n)
				return
			}
			b, err = io.ReadAll(obj.Reader)
			if err != nil {
				t.Error(err)
				return
			}
			if len(b) != tt.wantSize {
				t.Errorf("New() size = %v, wantSize = %v", len(b), tt.wantSize)
				return
			}
			n, err = obj.Reader.Seek(10, 0)
			if err != nil {
				t.Error(err)
				return
			}
			if n != 10 {
				t.Errorf("Expected 10, got %v", n)
				return
			}
			b, err = io.ReadAll(obj.Reader)
			if err != nil {
				return
			}
			if len(b) != tt.wantSize-10 {
				t.Errorf("New() size = %v, wantSize = %v", len(b), tt.wantSize)
				return
			}
			n, err = obj.Reader.Seek(10, io.SeekCurrent)
			if err != io.ErrUnexpectedEOF {
				t.Errorf("Expected io.ErrUnexpectedEOF, got %v", err)
				return
			}
			if n != 0 {
				t.Errorf("Expected 0, got %v", n)
				return
			}
		})
	}
}

func BenchmarkWithRandomData(b *testing.B) {
	type args struct {
		opts []Option
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "64KiB",
			args: args{opts: []Option{WithSize(1 << 16), WithRandomData().Apply()}},
		},
		{
			name: "1MiB",
			args: args{opts: []Option{WithSize(1 << 20), WithRandomData().Apply()}},
		},
		{
			name: "10MiB",
			args: args{opts: []Option{WithSize(10 << 20), WithRandomData().Apply()}},
		},
		{
			name: "10GiB",
			args: args{opts: []Option{WithSize(10 << 30), WithRandomData().Apply()}},
		},
	}
	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			got, err := New(tt.args.opts...)
			if err != nil {
				b.Errorf("New() error = %v", err)
				return
			}
			obj := got.Object()
			n, err := io.Copy(io.Discard, obj.Reader)
			if err != nil {
				b.Errorf("ioutil error = %v", err)
				return
			}
			b.SetBytes(n)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				obj = got.Object()
				_, err := io.Copy(io.Discard, obj.Reader)
				if err != nil {
					b.Errorf("New() error = %v", err)
					return
				}
			}
		})
	}
}

func TestSizeFn(t *testing.T) {
	fixed, err := SizeFn(WithRandomData().Apply(), WithSize(1000))
	if err != nil {
		t.Fatal(err)
	}
	rng := rand.New(rand.NewSource(1))
	if got := fixed(rng); got != 1000 {
		t.Fatalf("fixed size %d", got)
	}
	random, err := SizeFn(WithRandomData().Apply(), WithMinMaxSize(500, 5000), WithRandomSize(true))
	if err != nil {
		t.Fatal(err)
	}
	equal, err := SizeFn(WithRandomData().Apply(), WithMinMaxSize(4096, 4096), WithRandomSize(true))
	if err != nil {
		t.Fatal(err)
	}
	if got := equal(rng); got != 4096 {
		t.Errorf("equal bounds gave size %d, want 4096", got)
	}
	a, b := rand.New(rand.NewSource(7)), rand.New(rand.NewSource(7))
	for range 100 {
		x, y := random(a), random(b)
		if x != y {
			t.Fatal("same random source gave different sizes")
		}
		if x < 500 || x > 5000 {
			t.Fatalf("size %d outside 500-5000", x)
		}
	}
}
