package tinysl_test

import (
	"context"
	"sync"
	"testing"

	"github.com/andriiyaremenko/tinysl"
)

func BenchmarkGetTransinet(b *testing.B) {
	ctx := context.TODO()
	ctx, cancel := context.WithCancel(ctx)

	defer cancel()

	sl, _ := tinysl.Add(tinysl.Transient, nameServiceConstructor).ServiceLocator()

	for i := 0; i < b.N; i++ {
		_, _ = tinysl.Get[NameService](ctx, sl)
	}
}

func BenchmarkGetPerContext(b *testing.B) {
	sl, _ := tinysl.Add(tinysl.PerContext, nameServiceConstructor).ServiceLocator()

	for i := 0; i < b.N; i++ {
		runNCallsForPerContext[NameService](sl, 1)
	}
}

func BenchmarkGetPerContext2Services(b *testing.B) {
	sl, _ := tinysl.
		Add(tinysl.PerContext, nameServiceConstructor).
		Add(tinysl.PerContext, heroConstructor).
		ServiceLocator()

	for i := 0; i < b.N; i++ {
		runNCallsForPerContext[*Hero](sl, 1)
	}
}

func BenchmarkGetPerContext2Contexts(b *testing.B) {
	sl, _ := tinysl.Add(tinysl.PerContext, nameServiceConstructor).ServiceLocator()

	for i := 0; i < b.N; i++ {
		runNCallsForPerContext[NameService](sl, 2)
	}
}

func BenchmarkGetPerContext2Services2Contexts(b *testing.B) {
	sl, _ := tinysl.
		Add(tinysl.PerContext, nameServiceConstructor).
		Add(tinysl.PerContext, heroConstructor).
		ServiceLocator()

	for i := 0; i < b.N; i++ {
		runNCallsForPerContext[*Hero](sl, 2)
	}
}

func BenchmarkGetPerContext2Services10Contexts(b *testing.B) {
	sl, _ := tinysl.
		Add(tinysl.PerContext, nameServiceConstructor).
		Add(tinysl.PerContext, heroConstructor).
		ServiceLocator()

	for i := 0; i < b.N; i++ {
		runNCallsForPerContext[*Hero](sl, 2)
	}
}

func BenchmarkGetPerContext4Services10Contexts(b *testing.B) {
	sl, _ := tinysl.
		Add(tinysl.PerContext, nameServiceConstructor).
		Add(tinysl.PerContext, tableTimerConstructor).
		Add(tinysl.PerContext, heroConstructor).
		Add(tinysl.PerContext, impostorConstructor).
		ServiceLocator()

	for i := 0; i < b.N; i++ {
		runNCallsForPerContext[*Impostor](sl, 10)
	}
}

func BenchmarkGetPerContext4Services100Contexts(b *testing.B) {
	sl, _ := tinysl.
		Add(tinysl.PerContext, nameServiceConstructor).
		Add(tinysl.PerContext, tableTimerConstructor).
		Add(tinysl.PerContext, heroConstructor).
		Add(tinysl.PerContext, impostorConstructor).
		ServiceLocator()

	for i := 0; i < b.N; i++ {
		runNCallsForPerContext[*Impostor](sl, 100)
	}
}

func BenchmarkGetPerContext4Services1000Contexts(b *testing.B) {
	sl, _ := tinysl.
		Add(tinysl.PerContext, nameServiceConstructor).
		Add(tinysl.PerContext, tableTimerConstructor).
		Add(tinysl.PerContext, heroConstructor).
		Add(tinysl.PerContext, impostorConstructor).
		ServiceLocator()

	for i := 0; i < b.N; i++ {
		runNCallsForPerContext[*Impostor](sl, 1000)
	}
}

func BenchmarkGetPerContext4Services10_000Contexts(b *testing.B) {
	sl, _ := tinysl.
		Add(tinysl.PerContext, nameServiceConstructor).
		Add(tinysl.PerContext, tableTimerConstructor).
		Add(tinysl.PerContext, heroConstructor).
		Add(tinysl.PerContext, impostorConstructor).
		ServiceLocator()

	for i := 0; i < b.N; i++ {
		runNCallsForPerContext[*Impostor](sl, 10_000)
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

func BenchmarkGetSingleton(b *testing.B) {
	ctx := context.TODO()
	ctx, cancel := context.WithCancel(ctx)

	defer cancel()

	sl, _ := tinysl.Add(tinysl.Singleton, nameServiceConstructor).ServiceLocator()

	for i := 0; i < b.N; i++ {
		_, _ = tinysl.Get[NameService](ctx, sl)
	}
}
