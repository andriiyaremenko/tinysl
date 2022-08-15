package tinysl

import "reflect"

type propertyFiller struct {
	Type         reflect.Type
	Dependencies []string
	NewInstance  func(values ...any) (any, error)
}

// Type constructor that would automatically fill public fields using registered constructors.
func T[Type any]() (propertyFiller, error) {
	t := reflect.TypeOf(new(Type)).Elem()

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
		Type:         reflect.TypeOf(new(Type)).Elem(),
		Dependencies: dependencies,
		NewInstance:  getValueInstance[Type](fields),
	}, nil
}

// *Type constructor that would automatically fill public fields using registered constructors.
func P[Type any]() (propertyFiller, error) {
	t := reflect.TypeOf(new(Type)).Elem()

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
		Type:         reflect.TypeOf(new(Type)),
		Dependencies: dependencies,
		NewInstance:  getPointerInstance[Type](fields),
	}, nil
}

// Interface constructor that would use *Type as implementation
// and automatically fill public fields using registered constructors.
func I[Interface, Type any]() (propertyFiller, error) {
	p := reflect.TypeOf(new(Type))
	t := p.Elem()
	i := reflect.TypeOf(new(Interface)).Elem()

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
		Type:         reflect.TypeOf(new(Interface)).Elem(),
		Dependencies: dependencies,
		NewInstance:  getPointerInstance[Type](fields),
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
