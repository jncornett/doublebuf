package doublebuf

import (
	"context"
	"testing"
	"time"
)

func BenchmarkDoubleBuffer(b *testing.B) {
	b.Run("Next", func(b *testing.B) {
		db := New(0, 0)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		// feeder goroutine
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}
				for i := 0; i < b.N; i++ {
					j, err := db.Back(ctx)
					if err != nil {
						return
					}
					*j = i
					db.Ready()
				}
			}
		}()
		b.ReportAllocs()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				for {
					_, changed := db.Next()
					if changed {
						break
					}
				}
			}
		})
	})
	b.Run("Back", func(b *testing.B) {
		db := New(0, 0)
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		// eater goroutine
		go func() {
			// Go until cancelled, but only check every b.N times.
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}
				for i := 0; i < b.N; i++ {
					db.Next()
				}
			}
		}()
		b.ReportAllocs()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				db.Back(ctx)
				db.Ready()
			}
		})
	})
}
