package ffx

import (
	"context"
	"fmt"
	"sort"

	"go.uber.org/fx"
)

// Invoke helps to define ordered providers
type Invoke int

// Special is a type used to give keys to modules which
//
//	can't really be identified by the returned type
type Special int

// Settings is the container of the constructors used for DI
type Settings struct {
	// modules is a map of constructors for DI
	//
	// In most cases the index will be a reflect. Type of element returned by
	// the constructor, but for some 'constructors' it's hard to specify what's
	// the return type should be (or the constructor returns fx group)
	modules map[interface{}]fx.Option

	// invokes are separate from modules as they can't be referenced by return
	// type, and must be applied in correct order
	invokes map[Invoke]fx.Option

	logger fx.Printer
}

// StopFunc is used to stop the fx app
type StopFunc func(context.Context) error

// New builds and starts a
func New(ctx context.Context, opts ...Option) (StopFunc, error) {
	settings := Settings{
		modules: map[interface{}]fx.Option{},
		invokes: map[Invoke]fx.Option{},
	}

	// apply module options in the right order
	if err := Options(opts...)(&settings); err != nil {
		return nil, fmt.Errorf("applying node options failed: %w", err)
	}

	// gather constructors for fx.Options
	ctors := make([]fx.Option, 0, len(settings.modules))
	for _, opt := range settings.modules {
		ctors = append(ctors, opt)
	}

	invokes := make([]Invoke, 0, len(settings.invokes))
	for inv := range settings.invokes {
		invokes = append(invokes, inv)
	}

	sort.Slice(invokes, func(i, j int) bool {
		return invokes[i] < invokes[j]
	})

	invokeCtors := make([]fx.Option, len(invokes))
	for ii := range invokes {
		invokeCtors[ii] = settings.invokes[invokes[ii]]
	}

	logOpt := fx.Logger(fxprinter)
	if settings.logger != nil {
		logOpt = fx.Logger(settings.logger)
	}

	app := fx.New(
		fx.Options(ctors...),
		fx.Options(invokeCtors...),
		logOpt,
	)

	// TODO: we probably should have a 'firewall' for Closing signal
	//  on this context, and implement closing logic through lifecycles
	//  correctly
	if err := app.Start(ctx); err != nil {
		// comment fx.NopLogger few lines above for easier debugging
		return nil, fmt.Errorf("starting node: %w", err)
	}

	return app.Stop, nil
}
