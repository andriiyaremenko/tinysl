package tinysl_test

import (
	"context"
	"testing"

	"github.com/andriiyaremenko/tinysl"
)

func BenchmarkParallelGetSingleton(b *testing.B) {
	sl, _ := tinysl.Add(tinysl.Singleton, nameServiceConstructor).ServiceLocator()

	runNCallsInParallel[NameService](b, sl, 1)
}

func BenchmarkParallelGetTransinet(b *testing.B) {
	sl, _ := tinysl.Add(tinysl.Transient, nameServiceConstructor).ServiceLocator()

	runNCallsInParallel[NameService](b, sl, 1)
}

func BenchmarkParallelGetPerContext(b *testing.B) {
	sl, _ := tinysl.Add(tinysl.PerContext, nameServiceConstructor).ServiceLocator()

	runNCallsInParallel[NameService](b, sl, 1)
}

func BenchmarkParallelGetPerContext4Services10Contexts(b *testing.B) {
	sl, _ := tinysl.
		Add(tinysl.PerContext, nameServiceConstructor).
		Add(tinysl.PerContext, tableTimerConstructor).
		Add(tinysl.PerContext, heroConstructor).
		Add(tinysl.PerContext, impostorConstructor).
		ServiceLocator()

	runNCallsInParallel[*Impostor](b, sl, 10)
}

func BenchmarkParallelGetPerContext4Services1000Contexts(b *testing.B) {
	sl, _ := tinysl.
		Add(tinysl.PerContext, nameServiceConstructor).
		Add(tinysl.PerContext, tableTimerConstructor).
		Add(tinysl.PerContext, heroConstructor).
		Add(tinysl.PerContext, impostorConstructor).
		ServiceLocator()

	runNCallsInParallel[*Impostor](b, sl, 1000)
}

func BenchmarkParallelGetPerContext4Services10_000Contexts(b *testing.B) {
	sl, _ := tinysl.
		Add(tinysl.PerContext, nameServiceConstructor).
		Add(tinysl.PerContext, tableTimerConstructor).
		Add(tinysl.PerContext, heroConstructor).
		Add(tinysl.PerContext, impostorConstructor).
		ServiceLocator()

	runNCallsInParallel[*Impostor](b, sl, 10_000)
}

func BenchmarkParallelGetPerContext4Services100_000Contexts(b *testing.B) {
	sl, _ := tinysl.
		Add(tinysl.PerContext, nameServiceConstructor).
		Add(tinysl.PerContext, tableTimerConstructor).
		Add(tinysl.PerContext, heroConstructor).
		Add(tinysl.PerContext, impostorConstructor).
		ServiceLocator()

	runNCallsInParallel[*Impostor](b, sl, 100_000)
}

func BenchmarkParallelGetPerContext4Services500_000Contexts(b *testing.B) {
	sl, _ := tinysl.
		Add(tinysl.PerContext, nameServiceConstructor).
		Add(tinysl.PerContext, tableTimerConstructor).
		Add(tinysl.PerContext, heroConstructor).
		Add(tinysl.PerContext, impostorConstructor).
		ServiceLocator()

	runNCallsInParallel[*Impostor](b, sl, 500_000)
}

func BenchmarkParallelGetPerContext4Services1_000_000Contexts(b *testing.B) {
	sl, _ := tinysl.
		Add(tinysl.PerContext, nameServiceConstructor).
		Add(tinysl.PerContext, tableTimerConstructor).
		Add(tinysl.PerContext, heroConstructor).
		Add(tinysl.PerContext, impostorConstructor).
		ServiceLocator()

	runNCallsInParallel[*Impostor](b, sl, 1_000_000)
}

func runNCallsInParallel[T any](b *testing.B, sl tinysl.ServiceLocator, n int) {
	b.ResetTimer()
	b.SetParallelism(n)
	b.RunParallel(func(pb *testing.PB) {
		ctx := context.TODO()

		for pb.Next() {
			ctx, cancel := context.WithCancel(ctx)
			_, _ = tinysl.Get[T](ctx, sl)

			cancel()
		}
	})
}
