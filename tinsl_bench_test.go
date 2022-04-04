package tinysl_test

import (
	"context"
	"testing"

	"github.com/andriiyaremenko/tinysl"
)

func BenchmarkGetTransinet(b *testing.B) {
	ctx := context.TODO()
	ctx, cancel := context.WithCancel(ctx)

	defer cancel()

	sl, _ := tinysl.Add(tinysl.Transient, nameServiceConstructor).ServiceLocator()

	for i := 0; i < b.N; i++ {
		_, _ = tinysl.Get[nameService](ctx, sl)
	}
}

func BenchmarkGetPerContext(b *testing.B) {
	ctx := context.TODO()
	ctx, cancel := context.WithCancel(ctx)

	defer cancel()

	sl, _ := tinysl.Add(tinysl.PerContext, nameServiceConstructor).ServiceLocator()

	for i := 0; i < b.N; i++ {
		_, _ = tinysl.Get[nameService](ctx, sl)
	}
}

func BenchmarkGetSingleton(b *testing.B) {
	ctx := context.TODO()
	ctx, cancel := context.WithCancel(ctx)

	defer cancel()

	sl, _ := tinysl.Add(tinysl.Singleton, nameServiceConstructor).ServiceLocator()

	for i := 0; i < b.N; i++ {
		_, _ = tinysl.Get[nameService](ctx, sl)
	}
}
