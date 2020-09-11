package slang

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/spy16/sabre"
)

// Case implements the switch case construct.
func Case(scope sabre.Scope, args []sabre.Value) (sabre.Value, error) {
	if len(args) < 2 {
		return nil, errors.New("case requires at-least 2 args")
	}

	res, err := sabre.Eval(scope, args[0])
	if err != nil {
		return nil, err
	}

	if len(args) == 2 {
		return sabre.Eval(scope, args[1])
	}

	start := 1
	for ; start < len(args); start += 2 {
		val := args[start]
		if start+1 >= len(args) {
			return val, nil
		}

		if sabre.Compare(res, val) {
			return sabre.Eval(scope, args[start+1])
		}
	}

	return nil, fmt.Errorf("no matching clause for '%s'", res)
}

// MacroExpand is a wrapper around the sabre MacroExpand function that
// ignores the expanded bool flag.
func MacroExpand(scope sabre.Scope, f sabre.Value) (sabre.Value, error) {
	f, _, err := sabre.MacroExpand(scope, f)
	return f, err
}

// Throw converts args to strings and returns an error with all the strings
// joined.
func Throw(scope sabre.Scope, args ...sabre.Value) error {
	return errors.New(strings.Trim(MakeString(args...).String(), "\""))
}

// Realize realizes a sequence by continuously calling First() and Next()
// until the sequence becomes nil.
func Realize(seq sabre.Seq) *sabre.List {
	var vals []sabre.Value

	for seq != nil {
		v := seq.First()
		if v == nil {
			break
		}
		vals = append(vals, v)
		seq = seq.Next()
	}

	return &sabre.List{Values: vals}
}

// TypeOf returns the type information object for the given argument.
func TypeOf(v interface{}) sabre.Value {
	return sabre.ValueOf(reflect.TypeOf(v))
}

// Implements checks if given value implements the interface represented
// by 't'. Returns error if 't' does not represent an interface type.
func Implements(v interface{}, t sabre.Type) (bool, error) {
	if t.T.Kind() == reflect.Ptr {
		t.T = t.T.Elem()
	}

	if t.T.Kind() != reflect.Interface {
		return false, fmt.Errorf("type '%s' is not an interface type", t)
	}

	return reflect.TypeOf(v).Implements(t.T), nil
}

// ToType attempts to convert given sabre value to target type. Returns
// error if conversion not possible.
func ToType(val sabre.Value, to sabre.Type) (sabre.Value, error) {
	rv := reflect.ValueOf(val)
	if rv.Type().ConvertibleTo(to.T) || rv.Type().AssignableTo(to.T) {
		return sabre.ValueOf(rv.Convert(to.T).Interface()), nil
	}

	return nil, fmt.Errorf("cannot convert '%s' to '%s'", rv.Type(), to.T)
}

// ThreadFirst threads the expressions through forms by inserting result of
// eval as first argument to next expr.
func ThreadFirst(scope sabre.Scope, args []sabre.Value) (sabre.Value, error) {
	return threadCall(scope, args, false)
}

// ThreadLast threads the expressions through forms by inserting result of
// eval as last argument to next expr.
func ThreadLast(scope sabre.Scope, args []sabre.Value) (sabre.Value, error) {
	return threadCall(scope, args, true)
}

// MakeString returns stringified version of all args.
func MakeString(vals ...sabre.Value) sabre.Value {
	argc := len(vals)
	switch argc {
	case 0:
		return sabre.String("")

	case 1:
		nilVal := sabre.Nil{}
		if vals[0] == nilVal || vals[0] == nil {
			return sabre.String("")
		}

		return sabre.String(strings.Trim(vals[0].String(), "\""))

	default:
		var sb strings.Builder
		for _, v := range vals {
			sb.WriteString(strings.Trim(v.String(), "\""))
		}
		return sabre.String(sb.String())
	}
}

func threadCall(scope sabre.Scope, args []sabre.Value, last bool) (sabre.Value, error) {
	if len(args) == 0 {
		return nil, errors.New("at-least 1 argument required")
	}

	res, err := sabre.Eval(scope, args[0])
	if err != nil {
		return nil, err
	}

	for args = args[1:]; len(args) > 0; args = args[1:] {
		form := args[0]

		switch f := form.(type) {
		case *sabre.List:
			if last {
				f.Values = append(f.Values, res)
			} else {
				f.Values = append([]sabre.Value{f.Values[0], res}, f.Values[1:]...)
			}
			res, err = sabre.Eval(scope, f)

		case sabre.Invokable:
			res, err = f.Invoke(scope, res)

		default:
			return nil, fmt.Errorf("%s is not invokable", reflect.TypeOf(res))
		}

		if err != nil {
			return nil, err
		}
	}

	return res, nil
}

func isTruthy(v sabre.Value) bool {
	if v == nil || v == (sabre.Nil{}) {
		return false
	}

	if b, ok := v.(sabre.Bool); ok {
		return bool(b)
	}

	return true
}

func slangRange(args ...int) (Any, error) {
	var result []sabre.Value

	switch len(args) {
	case 1:
		result = slangRange1(args[0])
	case 2:
		result = slangRange2(args[0], args[1])
	case 3:
		result = slangRange3(args[0], args[1], args[2])
	}

	return &sabre.List{Values: result}, nil
}

func slangRange1(max int) []sabre.Value {

	result := make([]sabre.Value, 0, max)
	for i := 0; i < max; i++ {
		result = append(result, sabre.Int64(i))
	}
	return result
}

func slangRange2(min, max int) []sabre.Value {

	result := make([]sabre.Value, 0, max-min)
	for i := min; i < max; i++ {
		result = append(result, sabre.Int64(i))
	}
	return result
}

func slangRange3(min, max, step int) []sabre.Value {

	result := make([]sabre.Value, 0, max-min)
	for i := min; i < max; i += step {
		result = append(result, sabre.Int64(i))
	}
	return result
}

func slangMap(scope sabre.Scope, args []sabre.Value) (sabre.Value, error) {

	if len(args) < 2 {
		return nil, invalidArgNumberError{2, len(args)}
	}

	fn, err := sabre.Eval(scope, args[0])
	if err != nil {
		return nil, err
	}

	val, err := sabre.Eval(scope, args[1])
	if err != nil {
		return nil, err
	}

	seq, ok := val.(sabre.Seq)

	if !ok {
		return nil, fmt.Errorf("invalid type given; expected %s got %T",
			"sabre.Seq", val)
	}

	list := Realize(seq)

	result := make([]sabre.Value, 0, len(list.Values))
	for _, v := range list.Values {

		applied, err := fn.(sabre.Invokable).Invoke(scope, v)
		if err != nil {
			return nil, err
		}

		result = append(result, applied)
	}

	return &sabre.List{Values: result}, nil
}

func filter(scope sabre.Scope, args []sabre.Value) (sabre.Value, error) {

	if len(args) < 2 {
		return nil, &invalidArgNumberError{2, len(args)}
	}

	fn, err := sabre.Eval(scope, args[0])
	if err != nil {
		return nil, err
	}

	val, err := sabre.Eval(scope, args[1])
	if err != nil {
		return nil, err
	}

	seq, ok := val.(sabre.Seq)

	if !ok {
		return nil, fmt.Errorf("Invalid type given; expected %s instead got %T",
			"sabre.Seq", val)
	}

	list := Realize(seq)
	result := make([]sabre.Value, 0, len(list.Values))

	for _, v := range list.Values {

		applied, err := fn.(sabre.Invokable).Invoke(scope, v)
		if err != nil {
			return nil, err
		}

		if isTruthy(applied) {
			result = append(result, v)
		}

	}

	return &sabre.List{Values: result}, nil
}

func reduce(scope sabre.Scope, args []sabre.Value) (sabre.Value, error) {

	if len(args) < 2 {
		return nil, &invalidArgNumberError{2, len(args)}
	}

	// predicate function
	fn, err := sabre.Eval(scope, args[0])
	if err != nil {
		return nil, err
	}

	// determine the list position, if 3 argument given, the list position
	// index starting from 0 is 1; otherwise list on the 2
	listPos := 1

	if len(args) == 3 {
		listPos = 2
	}

	val, err := sabre.Eval(scope, args[listPos])
	if err != nil {
		return nil, err
	}

	seq, ok := val.(sabre.Seq)

	if !ok {
		return nil, fmt.Errorf("Invalid type given; expected %s instead got %T",
			"sabre.Seq", val)
	}

	list := Realize(seq)

	// determine the initial value
	var result sabre.Value

	// if initial value given
	if len(args) == 3 {
		result, err = sabre.Eval(scope, args[1])
		if err != nil {
			return nil, err
		}
	} else {
		// if not use the first elem from the list
		result = list.Values[0]
		list.Values = list.Values[1:]
	}

	for _, v := range list.Values {

		applied, err := fn.(sabre.Invokable).Invoke(scope, result, v)
		if err != nil {
			return nil, err
		}

		result = applied
	}

	return result, nil
}

func doSeq(scope sabre.Scope, args []sabre.Value) (sabre.Value, error) {

	arg1 := args[0]
	vecs, ok := arg1.(sabre.Vector)
	if !ok {
		return nil, fmt.Errorf("Invalid type")
	}

	list, err := vecs.Values[1].Eval(scope)
	if err != nil {
		return nil, err
	}

	symbol, ok := vecs.Values[0].(sabre.Symbol)
	if !ok {
		return nil, fmt.Errorf("invalid type; expected symbol")
	}

	for _, v := range list.(*sabre.List).Values {
		scope.Bind(symbol.Value, v)
		for _, body := range args[1:] {
			_, err := body.Eval(scope)
			if err != nil {
				return nil, err
			}
		}
	}

	return sabre.Nil{}, nil
}

func mutate(scope sabre.Scope, args []sabre.Value) (sabre.Value, error) {

	if len(args) < 2 {
		return nil, fmt.Errorf(
			"invalid number of arguments; expected %d got %d", 2, len(args))
	}

	symbol, ok := args[0].(sabre.Symbol)
	if !ok {
		return nil, fmt.Errorf("Expected symbol")
	}

	value, err := args[1].Eval(scope)
	if err != nil {
		return nil, err
	}

	scope.Bind(symbol.Value, value)
	return value, nil
}
