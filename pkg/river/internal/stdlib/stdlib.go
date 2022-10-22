// Package stdlib contains standard library functions exposed to River configs.
package stdlib

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/grafana/agent/pkg/river/internal/value"
)

// Functions returns the list of stdlib functions by name. The interface{}
// value is always a River-compatible function value, where functions have at
// least one non-error return value, with an optionally supported error return
// value as the second return value.
var Functions = map[string]interface{}{
	"env": os.Getenv,

	// concat is implemented as a raw function so it can bypass allocations
	// converting arguments into []interface{}. concat is optimized to allow it
	// to perform well when it is in the hot path for combining targets from many
	// other blocks.
	"concat": value.RawFunction(func(funcValue value.Value, args ...value.Value) (value.Value, error) {
		if len(args) == 0 {
			return value.Array(), nil
		}

		// finalSize is the final size of the resulting concatenated array. We type
		// check our arguments while computing what finalSize will be.
		var finalSize int
		for i, arg := range args {
			if arg.Type() != value.TypeArray {
				return value.Null, value.ArgError{
					Function: funcValue,
					Argument: arg,
					Index:    i,
					Inner: value.TypeError{
						Value:    arg,
						Expected: value.TypeArray,
					},
				}
			}

			finalSize += arg.Len()
		}

		// Optimization: if there's only one array, we can just return it directly.
		// This is done *after* the previous loop to ensure that args[0] is a River
		// array.
		if len(args) == 1 {
			return args[0], nil
		}

		raw := make([]value.Value, 0, finalSize)
		for _, arg := range args {
			for i := 0; i < arg.Len(); i++ {
				raw = append(raw, arg.Index(i))
			}
		}

		return value.Array(raw...), nil
	}),

	"json_decode": func(in string) (interface{}, error) {
		var res interface{}
		err := json.Unmarshal([]byte(in), &res)
		if err != nil {
			return nil, err
		}
		return res, nil
	},

	// objmerge(object, object)
	"objmerge": value.RawFunction(func(funcValue value.Value, args ...value.Value) (value.Value, error) {
		if len(args) != 2 {
			return value.Null, value.Error{
				Value: funcValue,
				Inner: fmt.Errorf("expected 2 args, got %d", len(args)),
			}
		}

		var (
			objA = args[0]
			objB = args[1]
		)

		if objA.Type() != value.TypeObject {
			return value.Null, value.TypeError{
				Value:    objA,
				Expected: value.TypeObject,
			}
		} else if objB.Type() != value.TypeObject {
			return value.Null, value.TypeError{
				Value:    objB,
				Expected: value.TypeObject,
			}
		}

		var (
			aKeys = objA.Keys()
			bKeys = objB.Keys()
		)

		keys := make(map[string]value.Value, len(aKeys)+len(bKeys))
		for _, key := range aKeys {
			keys[key], _ = objA.Key(key)
		}
		for _, key := range bKeys {
			keys[key], _ = objB.Key(key)
		}

		return value.Object(keys), nil
	}),

	// map(function, arr)
	"map": value.RawFunction(func(funcValue value.Value, args ...value.Value) (value.Value, error) {
		if len(args) != 2 {
			return value.Null, value.Error{
				Value: funcValue,
				Inner: fmt.Errorf("expected 2 args, got %d", len(args)),
			}
		}

		var (
			fnArg  = args[0]
			arrArg = args[1]
		)

		if fnArg.Type() != value.TypeFunction {
			return value.Null, value.TypeError{
				Value:    fnArg,
				Expected: value.TypeFunction,
			}
		} else if arrArg.Type() != value.TypeArray {
			return value.Null, value.TypeError{
				Value:    arrArg,
				Expected: value.TypeArray,
			}
		}

		res := make([]value.Value, 0, arrArg.Len())
		for i := 0; i < arrArg.Len(); i++ {
			// Call fnArg for each element in arrArg.
			orig := arrArg.Index(i)
			mapped, err := fnArg.Call(orig)
			if err != nil {
				// TODO(rfratto): what is the right error to return here?
				return value.Null, value.Error{
					Value: orig,
					Inner: err,
				}
			}

			res = append(res, mapped)
		}

		return value.Array(res...), nil
	}),
}
