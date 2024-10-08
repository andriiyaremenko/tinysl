package tinysl_test

import (
	"context"
	"sync"
	"testing"

	"github.com/andriiyaremenko/tinysl"
)

func BenchmarkTotalGetPerContext(b *testing.B) {
	sl, _ := tinysl.Add(tinysl.PerContext, nameServiceConstructor).ServiceLocator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runNCallsForPerContext[NameService](sl, 1)
	}
}

func BenchmarkTotalGetPerContext2Services(b *testing.B) {
	sl, _ := tinysl.
		Add(tinysl.PerContext, nameServiceConstructor).
		Add(tinysl.PerContext, heroConstructor).
		ServiceLocator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runNCallsForPerContext[*Hero](sl, 1)
	}
}

func BenchmarkTotalGetPerContext2Contexts(b *testing.B) {
	sl, _ := tinysl.Add(tinysl.PerContext, nameServiceConstructor).ServiceLocator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runNCallsForPerContext[NameService](sl, 2)
	}
}

func BenchmarkTotalGetPerContext4Services10Contexts(b *testing.B) {
	sl, _ := tinysl.
		Add(tinysl.PerContext, nameServiceConstructor).
		Add(tinysl.PerContext, tableTimerConstructor).
		Add(tinysl.PerContext, heroConstructor).
		Add(tinysl.PerContext, impostorConstructor).
		ServiceLocator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runNCallsForPerContext[*Impostor](sl, 10)
	}
}

func BenchmarkTotalGetPerContext4Services100Contexts(b *testing.B) {
	sl, _ := tinysl.
		Add(tinysl.PerContext, nameServiceConstructor).
		Add(tinysl.PerContext, tableTimerConstructor).
		Add(tinysl.PerContext, heroConstructor).
		Add(tinysl.PerContext, impostorConstructor).
		ServiceLocator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runNCallsForPerContext[*Impostor](sl, 100)
	}
}

func BenchmarkTotalGetPerContext4Services1000Contexts(b *testing.B) {
	sl, _ := tinysl.
		Add(tinysl.PerContext, nameServiceConstructor).
		Add(tinysl.PerContext, tableTimerConstructor).
		Add(tinysl.PerContext, heroConstructor).
		Add(tinysl.PerContext, impostorConstructor).
		ServiceLocator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runNCallsForPerContext[*Impostor](sl, 1000)
	}
}

func BenchmarkTotalGetPerContext4Services10_000Contexts(b *testing.B) {
	sl, _ := tinysl.
		Add(tinysl.PerContext, nameServiceConstructor).
		Add(tinysl.PerContext, tableTimerConstructor).
		Add(tinysl.PerContext, heroConstructor).
		Add(tinysl.PerContext, impostorConstructor).
		ServiceLocator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runNCallsForPerContext[*Impostor](sl, 10_000)
	}
}

func BenchmarkTotalGetPerContext4Services50_000Contexts(b *testing.B) {
	sl, _ := tinysl.
		Add(tinysl.PerContext, nameServiceConstructor).
		Add(tinysl.PerContext, tableTimerConstructor).
		Add(tinysl.PerContext, heroConstructor).
		Add(tinysl.PerContext, impostorConstructor).
		ServiceLocator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runNCallsForPerContext[*Impostor](sl, 50_000)
	}
}

func BenchmarkTotalGetPerContext4Services75_000Contexts(b *testing.B) {
	sl, _ := tinysl.
		Add(tinysl.PerContext, nameServiceConstructor).
		Add(tinysl.PerContext, tableTimerConstructor).
		Add(tinysl.PerContext, heroConstructor).
		Add(tinysl.PerContext, impostorConstructor).
		ServiceLocator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runNCallsForPerContext[*Impostor](sl, 75_000)
	}
}

func BenchmarkTotalGetPerContext4Services100_000Contexts(b *testing.B) {
	sl, _ := tinysl.
		Add(tinysl.PerContext, nameServiceConstructor).
		Add(tinysl.PerContext, tableTimerConstructor).
		Add(tinysl.PerContext, heroConstructor).
		Add(tinysl.PerContext, impostorConstructor).
		ServiceLocator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runNCallsForPerContext[*Impostor](sl, 100_000)
	}
}

func BenchmarkTotalGetPerContext4Services150_000Contexts(b *testing.B) {
	sl, _ := tinysl.
		Add(tinysl.PerContext, nameServiceConstructor).
		Add(tinysl.PerContext, tableTimerConstructor).
		Add(tinysl.PerContext, heroConstructor).
		Add(tinysl.PerContext, impostorConstructor).
		ServiceLocator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runNCallsForPerContext[*Impostor](sl, 150_000)
	}
}

func BenchmarkTotalGetPerContext4Services250_000Contexts(b *testing.B) {
	sl, _ := tinysl.
		Add(tinysl.PerContext, nameServiceConstructor).
		Add(tinysl.PerContext, tableTimerConstructor).
		Add(tinysl.PerContext, heroConstructor).
		Add(tinysl.PerContext, impostorConstructor).
		ServiceLocator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runNCallsForPerContext[*Impostor](sl, 250_000)
	}
}

func BenchmarkTotalGetPerContext4Services400_000Contexts(b *testing.B) {
	sl, _ := tinysl.
		Add(tinysl.PerContext, nameServiceConstructor).
		Add(tinysl.PerContext, tableTimerConstructor).
		Add(tinysl.PerContext, heroConstructor).
		Add(tinysl.PerContext, impostorConstructor).
		ServiceLocator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runNCallsForPerContext[*Impostor](sl, 400_000)
	}
}

func BenchmarkTotalGetPerContext4Services500_000Contexts(b *testing.B) {
	sl, _ := tinysl.
		Add(tinysl.PerContext, nameServiceConstructor).
		Add(tinysl.PerContext, tableTimerConstructor).
		Add(tinysl.PerContext, heroConstructor).
		Add(tinysl.PerContext, impostorConstructor).
		ServiceLocator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runNCallsForPerContext[*Impostor](sl, 500_000)
	}
}

func BenchmarkTotalGetPerContext4Services750_000Contexts(b *testing.B) {
	sl, _ := tinysl.
		Add(tinysl.PerContext, nameServiceConstructor).
		Add(tinysl.PerContext, tableTimerConstructor).
		Add(tinysl.PerContext, heroConstructor).
		Add(tinysl.PerContext, impostorConstructor).
		ServiceLocator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runNCallsForPerContext[*Impostor](sl, 750_000)
	}
}

func BenchmarkTotalGetPerContext4Services1_000_000Contexts(b *testing.B) {
	sl, _ := tinysl.
		Add(tinysl.PerContext, nameServiceConstructor).
		Add(tinysl.PerContext, tableTimerConstructor).
		Add(tinysl.PerContext, heroConstructor).
		Add(tinysl.PerContext, impostorConstructor).
		ServiceLocator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runNCallsForPerContext[*Impostor](sl, 1_000_000)
	}
}

func runNCallsForPerContext[T any](sl tinysl.ServiceLocator, n int) {
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			ctx := context.TODO()
			ctx, cancel := context.WithCancel(ctx)
			_, _ = tinysl.Get[T](ctx, sl)

			cancel()
			wg.Done()
		}()
	}

	wg.Wait()
}
