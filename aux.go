package api

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"strings"
)

// This constants will be used on the method above
var (
	responseWriterPtrType = reflect.TypeOf((*http.ResponseWriter)(nil))
	tesponseWriterType    = responseWriterPtrType.Elem()
	requestPtrType        = reflect.TypeOf((*http.Request)(nil))
	requestType           = requestPtrType.Elem()
	errorSliceType        = reflect.TypeOf(([]error)(nil))
	errorType             = errorSliceType.Elem()
	errorNilValue         = reflect.New(errorType).Elem()
)

// This method return true if the received type is an context type
// It means that it doesn't need to be mapped and will be present in the context
// It also return an error message if user used *http.ResponseWriter or used http.Request
// Context types include error and []error Types
func isContextType(resourceType reflect.Type) bool {
	// Test if user used *http.ResponseWriter insted of http.ResponseWriter
	if resourceType.AssignableTo(responseWriterPtrType) {
		log.Fatalf("You asked for %s when you should used %s", resourceType, tesponseWriterType)
	}
	// Test if user used http.Request insted of *http.Request
	if resourceType.AssignableTo(requestType) {
		log.Fatalf("You asked for %s when you should used %s", resourceType, requestPtrType)
	}
	// Test if user used ID insted of *ID
	if resourceType.AssignableTo(idType) {
		log.Fatalf("You asked for %s when you should used %s", idType, idPtrType)
	}

	return resourceType.AssignableTo(tesponseWriterType) ||
		resourceType.AssignableTo(requestPtrType) ||
		resourceType.AssignableTo(errorType) ||
		resourceType.AssignableTo(errorSliceType) ||
		resourceType == idPtrType
}

// Return the Ptr to the given Value if passed one of those types
// Struct, *Struct, []Struct or []*Struct
func validPtrOfValue(value reflect.Value) (reflect.Value, error) {

	// If its an pointer, extract the element of that pointer
	value = elemOfValue(value)

	// Create a new Ptr to the given Value Type
	ptrValue := reflect.New(value.Type())

	if value.Kind() == reflect.Slice {

		// TODO
		// Here we should test if this is an Slice by the Struct type

		if value.IsNil() || value.Len() == 0 {
			// If it is nil or it has nothing insid
			// return the empty Ptr to this Value type
			return ptrValue, nil
		} else {
			// If the slice is initialized and has elements inside it,
			// so insert these values in it

			// Add the given Slice Values to this new Ptr to Slice Value
			ptrValue.Elem().Set(value)

			return ptrValue, nil
		}
	}

	if value.Kind() == reflect.Struct {
		// Add the given Struct Value to this new Ptr to this Struct type
		ptrValue.Elem().Set(value)

		return ptrValue, nil
	}

	// Return error if the final value is not one of the valid types
	return reflect.Value{}, errors.New("You should pass an struct or an slice of structs")
}

// Return the Value of the Elem of the given Slice
func slicePtrToElemPtrValue(value reflect.Value) reflect.Value {

	value = elemOfValue(value)

	// Creates a new Ptr to Elem of the Type this Slice stores
	ptrToElemValue := reflect.New(elemOfType(value.Type().Elem()))

	if value.IsNil() || value.Len() == 0 {
		// If given Slice is null or it has nothing inside it
		// return the new empty Value of the Elem inside this slice
		return ptrToElemValue
	}

	// If this slice has an Elem inside it
	// and set the value of the first Elem inside this Slice

	ptrToElemValue.Elem().Set(value.Index(0))

	return ptrToElemValue

}

// If Value is a Ptr, return the Elem it points to
func elemOfValue(value reflect.Value) reflect.Value {

	if value.Kind() == reflect.Ptr {
		// It occours if an Struct has an Field that points itself
		// so it should never occours, and will be cauch by the check for Circular Dependency
		if !value.Elem().IsValid() {
			value = reflect.New(elemOfType(value.Type()))
		}
		return value.Elem()
	}
	return value
}

// If Type is a Ptr, return the Type of the Elem it points to
func elemOfType(t reflect.Type) reflect.Type {
	if t.Kind() == reflect.Ptr {
		return t.Elem()
	}
	return t
}

// If Type is a Slice, return the Type of the Elem it stores
func elemOfSliceType(t reflect.Type) reflect.Type {
	if t.Kind() == reflect.Slice {
		return t.Elem()
	}
	return t
}

// If Type is a Ptr, return the Type of the Elem it points to
// If Type is a Slice, return the Type of the Elem it stores
func mainElemOfType(t reflect.Type) reflect.Type {
	t = elemOfType(t)
	t = elemOfSliceType(t)
	t = elemOfType(t)
	return t

}

// If Type is not a Ptr, return the one Type that points to it
func ptrOfType(t reflect.Type) reflect.Type {
	if t.Kind() != reflect.Ptr {
		return reflect.PtrTo(t)
	}
	return t
}

// Check if this field is exported: fieldValue.CanSet()
// and if this field is valid fo create Resources: Structs or Slices of Structs
// Return the true if is an exported field of those of types
// Struct, *Struct, []Struct or []*Struct
func isValidValue(v reflect.Value) bool {
	if !v.CanSet() {
		return false
	}

	_, err := validPtrOfValue(v)
	if err != nil {
		return false
	}

	return true
}

// Return the true if the dependency is one of those types
// Interface, Struct, *Struct, []Struct or []*Struct
func isValidDependencyType(t reflect.Type) error {
	t = elemOfType(t)

	if t.Kind() == reflect.Interface || t.Kind() == reflect.Struct ||
		t.Kind() == reflect.Slice && t.Elem().Kind() == reflect.Struct {
		return nil
	}

	return fmt.Errorf("Type %s is not allowed as dependency", t)
}

// Return true if this Type is a Slice or Ptr to Slice
func isSliceType(t reflect.Type) bool {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Kind() == reflect.Slice
}

// Init methods should have no Output,
// it should alter the first argument as a pointer
// Or, at least, return itself
func isValidInit(method reflect.Method) error {

	// Test if Init method return itself and/or error types
	// just one of each type is accepted
	itself := false
	err := false

	//errorType := reflect.TypeOf(errors.New("432"))

	// Method Struct owner Type
	owner := mainElemOfType(method.Type.In(0))
	for i := 0; i < method.Type.NumOut(); i++ {
		t := mainElemOfType(method.Type.Out(i))
		if t == owner && !itself {
			itself = true
			continue
		}
		if t == errorType && !err {
			err = true
			continue
		}

		return fmt.Errorf("Resource %s has an invalid Init method %s. "+
			"It can't outputs %s \n", method.Type.In(0), method.Type, t)
	}

	return nil
}

// Return true if given StructField is an exported Field
// return false if is an unexported Field
func isExportedField(field reflect.StructField) bool {
	firstChar := string([]rune(field.Name)[0])
	return firstChar == strings.ToUpper(firstChar)
}

// Return a new empty Value for one of these Types
// Struct, Ptr to Struct, Slice, Ptr to Slice
func newEmptyValue(t reflect.Type) (reflect.Value, error) {

	// For Struct or Slice
	if t.Kind() == reflect.Struct || t.Kind() == reflect.Slice {
		return reflect.New(t), nil // A new Ptr to Struct of this type
	}
	// For Ptr to Struct or Ptr to Slice
	if t.Kind() == reflect.Ptr && (t.Elem().Kind() == reflect.Struct || t.Elem().Kind() == reflect.Slice) {
		return reflect.New(t.Elem()), nil // A new Ptr to Struct of this type
	}

	return reflect.Value{}, fmt.Errorf("Can't create an empty Value for type  %s", t)
}
