package ffx

import (
	"fmt"
	"reflect"

	"go.uber.org/fx"
)

// Option is a functional option which can be used with the New function to
// change how the node is constructed
//
// Options are applied in sequence
type Option func(*Settings) error

// Options groups multiple options into one
func Options(opts ...Option) Option {
	return func(s *Settings) error {
		for _, opt := range opts {
			if err := opt(s); err != nil {
				return err
			}
		}
		return nil
	}
}

// Error is a special option which returns an error when applied
func Error(err error) Option {
	return func(_ *Settings) error {
		return err
	}
}

// ApplyIf returns non-nil Option when the check func returns true
func ApplyIf(check func(s *Settings) bool, opts ...Option) Option {
	return func(s *Settings) error {
		if check(s) {
			return Options(opts...)(s)
		}
		return nil
	}
}

// If returns non-nil Option when the variable b is true
func If(b bool, opts ...Option) Option {
	return ApplyIf(func(s *Settings) bool {
		return b
	}, opts...)
}

// Override option changes constructor for a given type
func Override(typ, constructor interface{}) Option {
	return func(s *Settings) error {
		if i, ok := typ.(Invoke); ok {
			s.invokes[i] = fx.Invoke(constructor)
			return nil
		}

		if c, ok := typ.(Special); ok {
			s.modules[c] = fx.Provide(constructor)
			return nil
		}
		ctor := as(constructor, typ)
		rt := reflect.TypeOf(typ).Elem()

		s.modules[rt] = fx.Provide(ctor)
		return nil
	}
}

// Providers converts the constructors to fx.Providers based on their first output
func Providers(constructors ...interface{}) Option {
	return func(s *Settings) error {
		for _, constructor := range constructors {
			t := reflect.TypeOf(constructor)
			if t.Kind() != reflect.Func {
				return fmt.Errorf("expect func but got %v ", t.Kind())
			}
			out := t.Out(0)
			s.modules[out] = fx.Provide(constructor)
		}
		return nil
	}
}

// Unset removes the specific constructor from the Settings
func Unset(typ interface{}) Option {
	return func(s *Settings) error {
		if i, ok := typ.(Invoke); ok {
			delete(s.invokes, i)
			return nil
		}

		if c, ok := typ.(Special); ok {
			delete(s.modules, c)
			return nil
		}
		rt := reflect.TypeOf(typ).Elem()

		delete(s.modules, rt)
		return nil
	}
}

// Logger is used to set fx logger
func Logger(p fx.Printer) Option {
	return func(s *Settings) error {
		s.logger = p
		return nil
	}
}

// From defines a constructor (*T) -> func(t T) T {return t}
func From(typ interface{}) interface{} {
	rt := []reflect.Type{reflect.TypeOf(typ).Elem()}
	ft := reflect.FuncOf(rt, rt, false)
	return reflect.MakeFunc(ft, func(args []reflect.Value) (results []reflect.Value) {
		return args
	}).Interface()
}

// Populate directly uses fx.Populate and bind it onto a Invoke key
func Populate(key Invoke, target ...interface{}) Option {
	return func(s *Settings) error {
		s.invokes[key] = fx.Populate(target...)
		return nil
	}
}

// from go-ipfs
// as casts input constructor to a given interface (if a value is given, it
// wraps it into a constructor).
//
// Note: this method may look like a hack, and in fact it is one.
// This is here only because https://github.com/uber-go/fx/issues/673 wasn't
// released yet
//
// Note 2: when making changes here, make sure this method stays at
// 100% coverage. This makes it less likely it will be terribly broken
func as(in interface{}, as interface{}) interface{} {
	outType := reflect.TypeOf(as)

	if outType.Kind() != reflect.Ptr {
		panic("outType is not a pointer")
	}

	inType := reflect.TypeOf(in)

	if inType.Kind() != reflect.Func || inType.AssignableTo(outType.Elem()) {
		ctype := reflect.FuncOf(nil, []reflect.Type{outType.Elem()}, false)

		return reflect.MakeFunc(ctype, func(args []reflect.Value) (results []reflect.Value) {
			out := reflect.New(outType.Elem())
			out.Elem().Set(reflect.ValueOf(in))

			return []reflect.Value{out.Elem()}
		}).Interface()
	}

	ins := make([]reflect.Type, inType.NumIn())
	outs := make([]reflect.Type, inType.NumOut())

	for i := range ins {
		ins[i] = inType.In(i)
	}
	outs[0] = outType.Elem()
	for i := range outs[1:] {
		outs[i+1] = inType.Out(i + 1)
	}

	ctype := reflect.FuncOf(ins, outs, false)

	return reflect.MakeFunc(ctype, func(args []reflect.Value) (results []reflect.Value) {
		outs := reflect.ValueOf(in).Call(args)

		out := reflect.New(outType.Elem())
		if outs[0].Type().AssignableTo(outType.Elem()) {
			// Out: Iface = In: *Struct; Out: Iface = In: OtherIface
			out.Elem().Set(outs[0])
		} else {
			// Out: Iface = &(In: Struct)
			t := reflect.New(outs[0].Type())
			t.Elem().Set(outs[0])
			out.Elem().Set(t)
		}
		outs[0] = out.Elem()

		return outs
	}).Interface()
}
