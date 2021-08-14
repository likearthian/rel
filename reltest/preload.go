package reltest

import (
	"context"
	"reflect"
	"strings"

	"github.com/go-rel/rel"
)

type preload []*MockPreload

func (p *preload) register(ctxData ctxData, field string, queriers ...rel.Querier) *MockPreload {
	mp := &MockPreload{ctxData: ctxData, argField: field, argQuery: rel.Build("", queriers...)}
	*p = append(*p, mp)
	return mp
}

func (p preload) execute(ctx context.Context, records interface{}, field string, queriers ...rel.Querier) error {
	query := rel.Build("", queriers...)
	for _, mp := range p {
		if fetchContext(ctx) == mp.ctxData &&
			(mp.argRecords == nil || reflect.DeepEqual(mp.argRecords, records)) &&
			(mp.argRecordsType == "" || mp.argRecordsType == reflect.TypeOf(records).String()) &&
			matchQuery(mp.argQuery, query) {

			if mp.result != nil {
				var (
					target = asSlice(records, false)
					result = asSlice(mp.result, true)
					path   = strings.Split(field, ".")
				)

				execPreload(target, result, path)
			}

			return mp.retError
		}
	}

	panic("TODO: Query doesn't match")
}

// MockPreload asserts and simulate Delete function for test.
type MockPreload struct {
	ctxData        ctxData
	result         interface{}
	argRecords     interface{}
	argRecordsType string
	argField       string
	argQuery       rel.Query
	retError       error
}

// Result sets the result of preload.
func (mp *MockPreload) Result(result interface{}) {
	mp.result = result
}

// For expect calls for given record.
func (md *MockPreload) For(records interface{}) *MockPreload {
	md.argRecords = records
	return md
}

// ForType expect calls for given type.
// Type must include package name, example: `model.User`.
func (md *MockPreload) ForType(typ string) *MockPreload {
	md.argRecordsType = "*" + strings.TrimPrefix(typ, "*")
	return md
}

// Error sets error to be returned.
func (md *MockPreload) Error(err error) {
	md.retError = err
}

// ConnectionClosed sets this error to be returned.
func (md *MockPreload) ConnectionClosed() {
	md.Error(ErrConnectionClosed)
}

type slice interface {
	ReflectValue() reflect.Value
	Reset()
	Get(index int) *rel.Document
	Len() int
}

func asSlice(v interface{}, readonly bool) slice {
	var (
		sl slice
		rt = reflect.TypeOf(v)
	)

	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}

	if rt.Kind() == reflect.Slice {
		sl = rel.NewCollection(v, readonly)
	} else {
		sl = rel.NewDocument(v, readonly)
	}

	return sl
}

func execPreload(target slice, result slice, path []string) {
	type frame struct {
		index int
		doc   *rel.Document
	}

	var (
		mappedResult map[interface{}]reflect.Value
		stack        = make([]frame, target.Len())
	)

	// init stack
	for i := 0; i < len(stack); i++ {
		stack[i] = frame{index: 0, doc: target.Get(i)}
	}

	for len(stack) > 0 {
		var (
			n       = len(stack) - 1
			top     = stack[n]
			assocs  = top.doc.Association(path[top.index])
			hasMany = assocs.Type() == rel.HasMany
		)

		stack = stack[:n]

		if top.index == len(path)-1 {
			var (
				curr   slice
				rValue = assocs.ReferenceValue()
				fField = assocs.ForeignField()
			)

			if rValue == nil {
				continue
			}

			if hasMany {
				curr, _ = assocs.Collection()
			} else {
				curr, _ = assocs.Document()
			}

			curr.Reset()

			if mappedResult == nil {
				mappedResult = mapResult(result, fField, hasMany)
			}

			if rv, ok := mappedResult[rValue]; ok {
				curr.ReflectValue().Set(rv)
			}
		} else {
			if assocs.Type() == rel.HasMany {
				var (
					col, loaded = assocs.Collection()
				)

				if !loaded {
					continue
				}

				stack = append(stack, make([]frame, col.Len())...)
				for i := 0; i < col.Len(); i++ {
					stack[n+i] = frame{
						index: top.index + 1,
						doc:   col.Get(i),
					}
				}
			} else {
				if doc, loaded := assocs.Document(); loaded {
					stack = append(stack, frame{
						index: top.index + 1,
						doc:   doc,
					})
				}
			}
		}
	}
}

func mapResult(result slice, fField string, hasMany bool) map[interface{}]reflect.Value {
	var (
		mapResult = make(map[interface{}]reflect.Value)
	)

	for i := 0; i < result.Len(); i++ {
		var (
			doc       = result.Get(i)
			rv        = doc.ReflectValue()
			fValue, _ = doc.Value(fField)
		)

		if hasMany {
			if _, ok := mapResult[fValue]; !ok {
				mapResult[fValue] = reflect.MakeSlice(reflect.SliceOf(rv.Type()), 0, 0)
			}

			mapResult[fValue] = reflect.Append(mapResult[fValue], rv)
		} else {
			mapResult[fValue] = rv
		}
	}

	return mapResult
}
