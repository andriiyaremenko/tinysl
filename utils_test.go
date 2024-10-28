package tinysl_test

import (
	"context"
	"fmt"
	"io"
	"time"
)

type HelloService interface {
	Hello() string
}

type ServiceWithPublicFields struct {
	Dependency   NameService
	someProperty string
}

func (s ServiceWithPublicFields) SomeProperty() string {
	return s.someProperty
}

func (s *ServiceWithPublicFields) Hello() string {
	return "Hello " + s.Dependency.Name()
}

func (s ServiceWithPublicFields) Name() string {
	return s.Dependency.Name()
}

type NameService interface {
	Name() string
}

type NameProvider string

func (s NameProvider) Name() string {
	return string(s)
}

func nameServiceDecoratorConstructor(prefix string) func(NameService) NameService {
	return func(s NameService) NameService {
		return &NameServiceDecorator{NameService: s, prefix: prefix}
	}
}

type NameServiceDecorator struct {
	NameService NameService
	prefix      string
}

func (s *NameServiceDecorator) Name() string {
	return s.prefix + " " + s.NameService.Name()
}

type TableTimer struct {
	ctx         context.Context
	nameService NameService
}

type Impostor struct {
	hero *Hero
	name string
}

func (i *Impostor) Disguise() {
	i.hero = &Hero{i.name}
}

func (i *Impostor) Announce() string {
	return fmt.Sprintf("%s is our hero!", i.name)
}

func (i *Impostor) Name() string {
	return i.name
}

type Hero struct {
	name string
}

func (h *Hero) Announce() string {
	return fmt.Sprintf("%s is our hero!", h.name)
}

func (c *TableTimer) Countdown(w io.Writer, seconds int) {
	go func() {
		total := time.Second * time.Duration(seconds)
		ticker := time.NewTicker(time.Second)

		for {
			select {
			case <-c.ctx.Done():
				ticker.Stop()
				_, _ = w.Write([]byte("oops, looks like you broke it!"))

				return
			case <-ticker.C:
				total -= time.Second
				_, _ = fmt.Fprintf(w, "%s have %d seconds left", c.nameService.Name(), total)

				if total == 0 {
					ticker.Stop()

					return
				}
			}
		}
	}()
}

func nameProviderConstructor() (NameProvider, error) {
	return NameProvider("Bob"), nil
}

func nameServiceConstructor() (NameService, error) {
	return NameProvider("Bob"), nil
}

func tableTimerConstructor(ctx context.Context, nameService NameService) (*TableTimer, error) {
	return &TableTimer{ctx, nameService}, nil
}

func impostorConstructor(nameService NameService, hero *Hero) (*Impostor, error) {
	return &Impostor{name: nameService.Name(), hero: hero}, nil
}

func disguisedImpostorConstructor(impostor *Impostor) (*Hero, error) {
	return &Hero{name: impostor.Name()}, nil
}

func heroConstructor(nameService NameService) (*Hero, error) {
	return &Hero{nameService.Name()}, nil
}

func nameServiceConstructorWithCleanup(cleanup func()) func() (NameService, func(), error) {
	return func() (NameService, func(), error) {
		return NameProvider("bob"), cleanup, nil
	}
}

func heroConstructorWithCleanup(cleanup func()) func(nameService NameService) (*Hero, func(), error) {
	return func(nameService NameService) (*Hero, func(), error) {
		return &Hero{nameService.Name()}, cleanup, nil
	}
}

func scaredHeroConstructorWithCleanup(nameService NameService) (*Hero, error) {
	panic(fmt.Errorf("scared"))
}
