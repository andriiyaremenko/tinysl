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
		_, _ = tinysl.Get[NameService](ctx, sl)
	}
}

func BenchmarkGetPerContext(b *testing.B) {
	ctx := context.TODO()
	ctx, cancel := context.WithCancel(ctx)

	defer cancel()

	sl, _ := tinysl.Add(tinysl.PerContext, nameServiceConstructor).ServiceLocator()

	for i := 0; i < b.N; i++ {
		_, _ = tinysl.Get[NameService](ctx, sl)
	}
}

func BenchmarkGetPerContext2Services(b *testing.B) {
	ctx := context.TODO()
	ctx, cancel := context.WithCancel(ctx)

	defer cancel()

	sl, _ := tinysl.
		Add(tinysl.PerContext, nameServiceConstructor).
		Add(tinysl.PerContext, heroConstructor).
		ServiceLocator()

	for i := 0; i < b.N; i++ {
		_, _ = tinysl.Get[*Hero](ctx, sl)
	}
}

func BenchmarkGetPerContext2Contexts(b *testing.B) {
	ctx := context.TODO()
	ctx1, cancel1 := context.WithCancel(ctx)
	ctx2, cancel2 := context.WithCancel(ctx)

	defer cancel1()
	defer cancel2()

	sl, _ := tinysl.Add(tinysl.PerContext, nameServiceConstructor).ServiceLocator()

	for i := 0; i < b.N; i++ {
		_, _ = tinysl.Get[NameService](ctx1, sl)
		_, _ = tinysl.Get[NameService](ctx2, sl)
	}
}

func BenchmarkGetPerContext2Services2Contexts(b *testing.B) {
	ctx := context.TODO()
	ctx1, cancel1 := context.WithCancel(ctx)
	ctx2, cancel2 := context.WithCancel(ctx)

	defer cancel1()
	defer cancel2()

	sl, _ := tinysl.
		Add(tinysl.PerContext, nameServiceConstructor).
		Add(tinysl.PerContext, heroConstructor).
		ServiceLocator()

	for i := 0; i < b.N; i++ {
		_, _ = tinysl.Get[*Hero](ctx1, sl)
		_, _ = tinysl.Get[*Hero](ctx2, sl)
	}
}

func BenchmarkGetPerContext2Services10Contexts(b *testing.B) {
	ctx := context.TODO()
	ctx1, cancel1 := context.WithCancel(ctx)
	ctx2, cancel2 := context.WithCancel(ctx)
	ctx3, cancel3 := context.WithCancel(ctx)
	ctx4, cancel4 := context.WithCancel(ctx)
	ctx5, cancel5 := context.WithCancel(ctx)
	ctx6, cancel6 := context.WithCancel(ctx)
	ctx7, cancel7 := context.WithCancel(ctx)
	ctx8, cancel8 := context.WithCancel(ctx)
	ctx9, cancel9 := context.WithCancel(ctx)
	ctx10, cancel10 := context.WithCancel(ctx)

	defer cancel1()
	defer cancel2()
	defer cancel3()
	defer cancel4()
	defer cancel5()
	defer cancel6()
	defer cancel7()
	defer cancel8()
	defer cancel9()
	defer cancel10()

	sl, _ := tinysl.
		Add(tinysl.PerContext, nameServiceConstructor).
		Add(tinysl.PerContext, heroConstructor).
		ServiceLocator()

	for i := 0; i < b.N; i++ {
		_, _ = tinysl.Get[*Hero](ctx1, sl)
		_, _ = tinysl.Get[*Hero](ctx2, sl)
		_, _ = tinysl.Get[*Hero](ctx3, sl)
		_, _ = tinysl.Get[*Hero](ctx4, sl)
		_, _ = tinysl.Get[*Hero](ctx5, sl)
		_, _ = tinysl.Get[*Hero](ctx6, sl)
		_, _ = tinysl.Get[*Hero](ctx7, sl)
		_, _ = tinysl.Get[*Hero](ctx8, sl)
		_, _ = tinysl.Get[*Hero](ctx9, sl)
		_, _ = tinysl.Get[*Hero](ctx10, sl)
	}
}

func BenchmarkGetPerContext4Services10Contexts(b *testing.B) {
	ctx := context.TODO()
	ctx1, cancel1 := context.WithCancel(ctx)
	ctx2, cancel2 := context.WithCancel(ctx)
	ctx3, cancel3 := context.WithCancel(ctx)
	ctx4, cancel4 := context.WithCancel(ctx)
	ctx5, cancel5 := context.WithCancel(ctx)
	ctx6, cancel6 := context.WithCancel(ctx)
	ctx7, cancel7 := context.WithCancel(ctx)
	ctx8, cancel8 := context.WithCancel(ctx)
	ctx9, cancel9 := context.WithCancel(ctx)
	ctx10, cancel10 := context.WithCancel(ctx)

	defer cancel1()
	defer cancel2()
	defer cancel3()
	defer cancel4()
	defer cancel5()
	defer cancel6()
	defer cancel7()
	defer cancel8()
	defer cancel9()
	defer cancel10()

	sl, _ := tinysl.
		Add(tinysl.PerContext, nameServiceConstructor).
		Add(tinysl.PerContext, tableTimerConstructor).
		Add(tinysl.PerContext, heroConstructor).
		Add(tinysl.PerContext, impostorConstructor).
		ServiceLocator()

	for i := 0; i < b.N; i++ {
		_, _ = tinysl.Get[*Impostor](ctx1, sl)
		_, _ = tinysl.Get[*Impostor](ctx2, sl)
		_, _ = tinysl.Get[*Impostor](ctx3, sl)
		_, _ = tinysl.Get[*Impostor](ctx4, sl)
		_, _ = tinysl.Get[*Impostor](ctx5, sl)
		_, _ = tinysl.Get[*Impostor](ctx6, sl)
		_, _ = tinysl.Get[*Impostor](ctx7, sl)
		_, _ = tinysl.Get[*Impostor](ctx8, sl)
		_, _ = tinysl.Get[*Impostor](ctx9, sl)
		_, _ = tinysl.Get[*Impostor](ctx10, sl)
	}
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
