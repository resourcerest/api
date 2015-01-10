package resource

import (
	"log"
	"net/http"
	"reflect"
)

type Context struct {
	Handler *Handler
	Values  []reflect.Value
	IDMap   IDMap
}

// Creates a new context
// It creates the initial state used to answer a request
// Since states are not allowed to be stored on te server,
// the request state is all the service has to answer a request
func newContext(handler *Handler, w http.ResponseWriter, req *http.Request, idMap IDMap) *Context {
	return &Context{
		Handler: handler,
		Values: []reflect.Value{
			reflect.ValueOf(w),
			reflect.ValueOf(req),
		},
		IDMap: idMap,
	}
}

func (c *Context) run() []reflect.Value {

	log.Println("Running Context Handler Method:", c.Handler.Method.Method.Type)

	// Then run the main method
	// c.Method.Input[0] = the Method Resource Type
	inputs := c.getInputs(c.Handler.Method)

	return c.Handler.Method.Method.Func.Call(inputs)
}

// Return the inputs from a list of requested types
// For the especial case of the ID input, we should know the requesterType
func (c *Context) getInputs(m *Method) []reflect.Value {

	inputsTypes := m.Inputs

	requesterType := m.Owner

	inputs := make([]reflect.Value, len(inputsTypes))

	log.Println("Getting inputs:", inputsTypes)

	for i, t := range inputsTypes {

		//log.Println("Getting input", t)
		inputs[i] = c.valueOf(t, requesterType)
		//log.Println("Getted", inputs[i], "for", t)

		// If the input isn't a pointer, we have to transform in an element
		// Especial ID case should not be treated
		if t.Kind() != reflect.Ptr && t != IDType {
			inputs[i] = inputs[i].Elem()
			//log.Println("Transformed", inputs[i], "for", t)
		}

	}

	//log.Println("Returning inputs:", inputs, "for", inputsTypes)

	return inputs
}

// Get the reflect.Value for the Interface
// it will ever exist
func (c *Context) valueOf(t reflect.Type, requesterType reflect.Type) reflect.Value {

	log.Println("Searching for", t)

	if t.Kind() == reflect.Interface {
		return c.interfaceValue(t)
	}

	// It's an struct

	// Especial case for ID request
	if t == IDType {
		return c.idValue(requesterType)
	}

	// NonPointer Struct and Slices cases
	if t.Kind() == reflect.Struct || t.Kind() == reflect.Slice {
		return c.nonPtrValue(t)
	}

	if t.Kind() == reflect.Ptr {
		return c.ptrValue(t)
	}

	// It should never occours,
	// cause it should be treated on the mapping time
	log.Panicf("Depenency type %s of %s not accepted",
		"and not treated on the method mapping time\n", t.Kind(), t)

	return reflect.Value{}
}

// Get the reflect.Value for the Interface
func (c *Context) interfaceValue(t reflect.Type) reflect.Value {

	for _, v := range c.Values {
		if v.Type().Implements(t) {
			return v
		}
	}

	// If this value doesn't exist yet, so initialie it
	return c.initDependencie(t)
}

// Get the reflect.Value for the Struct
func (c *Context) nonPtrValue(t reflect.Type) reflect.Value {

	for _, v := range c.Values {
		if v.Type().Elem() == t {
			return v
		}
	}

	// If this value doesn't exist yet, so initialie it
	return c.initDependencie(t)
}

// Get the reflect.Value for the Ptr to Struct
func (c *Context) ptrValue(t reflect.Type) reflect.Value {

	for _, v := range c.Values {
		if v.Type() == t {
			return v
		}
	}

	// If this value doesn't exist yet, so initialie it
	return c.initDependencie(t)
}

// Get the reflect.Value for the ID list caught in the URI
// It returns an empty ID if ID were not passed in the URI
func (c *Context) idValue(t reflect.Type) reflect.Value {

	id, exist := c.IDMap[t]
	if exist {
		return id // its an reflect.Value from the type of ID
	}

	// Doesn't exist, returning an empty default ID
	return EmptyIDValue
}

//
// --------------------------- not used
//

// Construct all the dependencies level by level
// Garants that every dependencie exists before be requisited
func (c *Context) initDependencie(t reflect.Type) reflect.Value {

	dependencie, exist := c.Handler.Dependencies[t]
	if !exist { // It should never occours
		log.Panicf("Dependencie %s not mapped!!!", t)
	}

	log.Println("Constructing dependency", dependencie.Value.Type())

	// This Value will be mapped in the index index
	index := len(c.Values)

	c.Values = append(c.Values, dependencie.Value)

	if dependencie.Method != nil {

		inputs := c.getInputs(dependencie.Method) //dependencie.Input, dependencie.Value.Type())

		out := make([]reflect.Value, dependencie.Method.Method.Type.NumOut())

		log.Printf("Calling %s with %q \n", dependencie.Method.Method.Type, inputs)

		out = dependencie.Method.Method.Func.Call(inputs)

		// If the Init method return something,
		// it will be the resource itself with
		// its values updated
		if dependencie.Method.NumOut > 0 {

			log.Println("Replacing Initial value of", c.Values[index])

			c.Values[index] = out[0]
		}
	}

	log.Println("Constructed", c.Values[index], "for", t, "value", c.Values[index].Interface())

	return c.Values[index]
}
