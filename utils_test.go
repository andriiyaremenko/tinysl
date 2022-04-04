package tinysl_test

import (
	"context"
	"fmt"
	"io"
	"time"
)

type nameService interface {
	Name() string
}

type nameProvider string

func (s nameProvider) Name() string {
	return string(s)
}

type tableTimer struct {
	ctx         context.Context
	nameService nameService
}

type impostor struct {
	name string
	hero *hero
}

func (i *impostor) Disguise() {
	i.hero = &hero{i.name}
}

func (i *impostor) Announce() string {
	return fmt.Sprintf("%s is our hero!", i.name)
}

func (i *impostor) Name() string {
	return i.name
}

type hero struct {
	name string
}

func (h *hero) Announce() string {
	return fmt.Sprintf("%s is our hero!", h.name)
}

func (c *tableTimer) Countdown(w io.Writer, seconds int) {
	go func() {
		total := time.Second * time.Duration(seconds)
		ticker := time.NewTicker(time.Second)

		for {
			select {
			case <-c.ctx.Done():
				ticker.Stop()
				w.Write([]byte("oops, looks like you broke it!"))

				return
			case <-ticker.C:
				total -= time.Second
				w.Write([]byte(fmt.Sprintf("%s have %d seconds left", c.nameService.Name(), total)))
				if total == 0 {
					ticker.Stop()

					return
				}
			}
		}
	}()
}

func nameProviderConstructor() (nameProvider, error) {
	return nameProvider("Bob"), nil
}

func nameServiceConstructor() (nameService, error) {
	return nameProvider("Bob"), nil
}

func tableTimerConstructor(ctx context.Context, nameService nameService) (*tableTimer, error) {
	return &tableTimer{ctx, nameService}, nil
}

func impostorConstructor(nameService nameService, hero *hero) (*impostor, error) {
	return &impostor{name: nameService.Name(), hero: hero}, nil
}

func disguisedImpostorConstructor(impostor *impostor) (*hero, error) {
	return &hero{name: impostor.Name()}, nil
}

func heroConstructor(nameService nameService) (*hero, error) {
	return &hero{nameService.Name()}, nil
}
