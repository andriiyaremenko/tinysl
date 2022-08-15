package tinysl

import "reflect"

type propertyFiller struct {
	Type         reflect.Type
	Dependencies []string
	NewInstance  func(values ...any) (any, error)
}

func T[T any]() (propertyFiller, error) {
	t := reflect.TypeOf(new(T)).Elem()

	if t.Kind() != reflect.Struct {
		return propertyFiller{}, &TError{T: t}
	}

	filedIndex := 0
	fields := make(map[int]int)
	dependencies := make([]string, 0, 1)

	for i := 0; i < t.NumField(); i++ {
		if !t.Field(i).IsExported() {
			continue
		}

		dependencies = append(dependencies, t.Field(i).Type.String())
		fields[filedIndex] = i
		filedIndex++
	}

	return propertyFiller{
		Type:         reflect.TypeOf(new(T)).Elem(),
		Dependencies: dependencies,
		NewInstance:  getValueInstance[T](fields),
	}, nil
}

func P[T any]() (propertyFiller, error) {
	t := reflect.TypeOf(new(T)).Elem()

	if t.Kind() != reflect.Struct {
		return propertyFiller{}, &PError{T: t}
	}

	filedIndex := 0
	fields := make(map[int]int)
	dependencies := make([]string, 0, 1)

	for i := 0; i < t.NumField(); i++ {
		if !t.Field(i).IsExported() {
			continue
		}

		dependencies = append(dependencies, t.Field(i).Type.String())
		fields[filedIndex] = i
		filedIndex++
	}

	return propertyFiller{
		Type:         reflect.TypeOf(new(T)),
		Dependencies: dependencies,
		NewInstance:  getPointerInstance[T](fields),
	}, nil
}

func I[I, T any]() (propertyFiller, error) {
	p := reflect.TypeOf(new(T))
	t := p.Elem()
	i := reflect.TypeOf(new(I)).Elem()

	if t.Kind() != reflect.Struct {
		return propertyFiller{}, newIError(ErrIWrongTType, i, t)
	}

	if i.Kind() != reflect.Interface {
		return propertyFiller{}, newIError(ErrIWrongIType, i, t)
	}

	if !p.Implements(i) {
		return propertyFiller{}, newIError(ErrITDoesNotImplementI, i, t)
	}

	filedIndex := 0
	fields := make(map[int]int)
	dependencies := make([]string, 0, 1)

	for i := 0; i < t.NumField(); i++ {
		if !t.Field(i).IsExported() {
			continue
		}

		dependencies = append(dependencies, t.Field(i).Type.String())
		fields[filedIndex] = i
		filedIndex++
	}

	return propertyFiller{
		Type:         reflect.TypeOf(new(I)).Elem(),
		Dependencies: dependencies,
		NewInstance:  getPointerInstance[T](fields),
	}, nil
}

func getValueInstance[T any](fields map[int]int) func(...any) (any, error) {
	return func(values ...any) (any, error) {
		p := reflect.ValueOf(new(T)).Elem()

		for i, v := range values {
			p.Field(fields[i]).Set(reflect.ValueOf(v))
		}

		return p.Interface(), nil
	}
}

func getPointerInstance[T any](fields map[int]int) func(...any) (any, error) {
	return func(values ...any) (any, error) {
		p := reflect.ValueOf(new(T)).Elem()

		for i, v := range values {
			p.Field(fields[i]).Set(reflect.ValueOf(v))
		}

		return p.Addr().Interface(), nil
	}
}
