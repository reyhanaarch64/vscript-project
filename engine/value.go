package engine

import (
	"fmt"
	"math"
)

type ReturnSignal struct{ Value interface{} }

type Function struct {
	Params  []string
	Body    *BlockStmt
	Closure *Env
}

type Media struct {
	Path string
	Kind string
	Size int64
}

type Env struct {
	values map[string]interface{}
	parent *Env
}

func NewEnv(parent *Env) *Env                    { return &Env{values: map[string]interface{}{}, parent: parent} }
func (e *Env) Define(name string, v interface{}) { e.values[name] = v }
func (e *Env) Get(name string) (interface{}, bool) {
	if v, ok := e.values[name]; ok {
		return v, true
	}
	if e.parent != nil {
		return e.parent.Get(name)
	}
	return nil, false
}
func (e *Env) Set(name string, v interface{}) bool {
	if _, ok := e.values[name]; ok {
		e.values[name] = v
		return true
	}
	if e.parent != nil {
		return e.parent.Set(name, v)
	}
	return false
}

func number(v interface{}) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case int:
		return float64(x)
	case int64:
		return float64(x)
	case float32:
		return float64(x)
	case bool:
		if x {
			return 1
		}
		return 0
	default:
		return 0
	}
}
func truth(v interface{}) bool {
	if v == nil {
		return false
	}
	switch x := v.(type) {
	case bool:
		return x
	case float64:
		return x != 0
	case string:
		return x != ""
	}
	return true
}
func formatValue(v interface{}) string {
	switch x := v.(type) {
	case string:
		return x
	case float64:
		if math.Trunc(x) == x {
			return fmt.Sprintf("%.0f", x)
		}
		return fmt.Sprintf("%g", x)
	case nil:
		return "null"
	default:
		return fmt.Sprint(x)
	}
}
