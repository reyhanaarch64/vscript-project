package engine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Renderer struct {
	W, H       int
	FPS        int
	Duration   float64
	imageCache map[string]image.Image
}

func Run(source, filename string) error {
	lx := NewLexer(source)
	tokens, err := lx.Lex()
	if err != nil {
		return err
	}
	p := NewParser(tokens)
	program, err := p.Parse()
	if err != nil {
		return err
	}
	return execute(program, filename)
}

func execute(program *Program, filename string) error {
	env := NewEnv(nil)
	registerBuiltins(env)
	env.Define("__file", filename)
	for _, stmt := range program.Statements {
		if _, ok := stmt.(ExportStmt); ok {
			continue
		}
		if _, ok := stmt.(VideoStmt); ok {
			continue
		}
		if err := execStmt(stmt, env); err != nil {
			return err
		}
	}
	var videos []*VideoPlan
	for _, stmt := range program.Statements {
		if v, ok := stmt.(VideoStmt); ok {
			wv, e := eval(v.Width, env)
			if e != nil {
				return e
			}
			hv, e := eval(v.Height, env)
			if e != nil {
				return e
			}
			fv, e := eval(v.FPS, env)
			if e != nil {
				return e
			}
			dv, e := eval(v.Duration, env)
			if e != nil {
				return e
			}
			w := int(number(wv))
			h := int(number(hv))
			fps := int(number(fv))
			dur := durationValue(dv)
			dw, dh := w, h
			if v.DesignWidth != nil && v.DesignHeight != nil {
				dwv, de := eval(v.DesignWidth, env)
				if de != nil {
					return de
				}
				dhv, de := eval(v.DesignHeight, env)
				if de != nil {
					return de
				}
				dw = int(number(dwv))
				dh = int(number(dhv))
			}
			videos = append(videos, &VideoPlan{Width: w, Height: h, FPS: fps, Duration: dur, DesignWidth: dw, DesignHeight: dh, Body: v.Body, Env: env})
		}
	}
	for _, v := range videos {
		env.Define("lastVideo", v)
	}
	for _, stmt := range program.Statements {
		ex, ok := stmt.(ExportStmt)
		if !ok {
			continue
		}
		targetValue, err := eval(ex.Target, env)
		if err != nil {
			return err
		}
		target := formatValue(targetValue)
		var bitrate int64 = 4000000
		if ex.Bitrate != nil {
			bv, e := eval(ex.Bitrate, env)
			if e != nil {
				return e
			}
			bitrate = int64(number(bv))
		}
		if len(videos) == 0 {
			return fmt.Errorf("%s: tidak ada blok video untuk diekspor", filename)
		}
		v := videos[len(videos)-1]
		if ex.Audio == nil {
			if err := v.Render(target, ex.Codec, bitrate); err != nil {
				return fmt.Errorf("%s: %w", filename, err)
			}
			continue
		}
		av, err := eval(ex.Audio, env)
		if err != nil {
			return err
		}
		audioPath := mediaPath(av)
		if audioPath == "" {
			return fmt.Errorf("%s: sumber audio tidak valid", filename)
		}
		tmpFile, err := os.CreateTemp(filepath.Dir(target), ".vscript-video-*.mp4")
		if err != nil {
			return fmt.Errorf("%s: gagal membuat file sementara audio: %w", filename, err)
		}
		tmpPath := tmpFile.Name()
		tmpFile.Close()
		os.Remove(tmpPath)
		if err := v.Render(tmpPath, ex.Codec, bitrate); err != nil {
			os.Remove(tmpPath)
			return fmt.Errorf("%s: %w", filename, err)
		}
		if err := muxAudio(tmpPath, audioPath, target); err != nil {
			os.Remove(tmpPath)
			return fmt.Errorf("%s: %w", filename, err)
		}
		os.Remove(tmpPath)
	}
	return nil
}

func mediaPath(v interface{}) string {
	switch x := v.(type) {
	case Media:
		return x.Path
	case *Media:
		if x != nil {
			return x.Path
		}
	default:
		return formatValue(v)
	}
	return ""
}

func durationValue(v interface{}) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case string:
		return parseDurationText(x)
	default:
		return number(v)
	}
}
func parseDurationText(s string) float64 {
	s = strings.TrimSpace(strings.ToLower(s))
	var n float64
	if strings.HasSuffix(s, "ms") {
		fmt.Sscanf(strings.TrimSuffix(s, "ms"), "%f", &n)
		return n / 1000
	}
	if strings.HasSuffix(s, "s") {
		fmt.Sscanf(strings.TrimSuffix(s, "s"), "%f", &n)
		return n
	}
	fmt.Sscanf(s, "%f", &n)
	return n
}

func execStmt(stmt Stmt, env *Env) error {
	switch s := stmt.(type) {
	case BlockStmt:
		return execBlock(&s, env)
	case *BlockStmt:
		return execBlock(s, env)
	case ExprStmt:
		_, err := eval(s.Expr, env)
		return err
	case LetStmt:
		v, err := eval(s.Value, env)
		if err != nil {
			return err
		}
		env.Define(s.Name, v)
		return nil
	case AssignStmt:
		v, err := eval(s.Value, env)
		if err != nil {
			return err
		}
		if !env.Set(s.Name, v) {
			env.Define(s.Name, v)
		}
		return nil
	case FunctionStmt:
		env.Define(s.Name, &Function{Params: s.Params, Body: s.Body, Closure: env})
		return nil
	case BreakStmt:
		panic(BreakSignal{})
	case ContinueStmt:
		panic(ContinueSignal{})
	case ReturnStmt:
		v, err := eval(s.Value, env)
		if err != nil {
			return err
		}
		panic(ReturnSignal{v})
	case IfStmt:
		c, err := eval(s.Cond, env)
		if err != nil {
			return err
		}
		if truth(c) {
			return execBlock(s.Then, env)
		}
		if s.Else != nil {
			return execBlock(s.Else, env)
		}
		return nil
	case ForStmt:
		startValue, err := eval(s.Start, env)
		if err != nil {
			return err
		}
		endValue, err := eval(s.End, env)
		if err != nil {
			return err
		}
		start := int(number(startValue))
		end := int(number(endValue))
		step := 1
		if start > end {
			step = -1
		}
		for i := start; ; i += step {
			env.Define(s.Name, float64(i))
			control, loopErr := executeLoopBody(s.Body, env)
			if loopErr != nil {
				return loopErr
			}
			if control == 2 {
				break
			}
			if i == end {
				break
			}
		}
		return nil
	case WhileStmt:
		for {
			cond, err := eval(s.Cond, env)
			if err != nil {
				return err
			}
			if !truth(cond) {
				break
			}
			control, loopErr := executeLoopBody(s.Body, env)
			if loopErr != nil {
				return loopErr
			}
			if control == 2 {
				break
			}
		}
		return nil
	default:
		return nil
	}
}
func executeLoopBody(body *BlockStmt, env *Env) (control int, err error) {
	defer func() {
		if r := recover(); r != nil {
			switch x := r.(type) {
			case BreakSignal:
				control, err = 2, nil
			case ContinueSignal:
				control, err = 1, nil
			case loopError:
				control, err = -1, x.err
			default:
				panic(r)
			}
		}
	}()
	if blockErr := execBlock(body, env); blockErr != nil {
		panic(loopError{blockErr})
	}
	return 0, nil
}

type loopError struct{ err error }

func execBlock(b *BlockStmt, env *Env) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if rs, ok := r.(ReturnSignal); ok {
				panic(rs)
			}
			panic(r)
		}
	}()
	for _, s := range b.Statements {
		if err := execStmt(s, env); err != nil {
			return err
		}
	}
	return nil
}

func eval(expr Expr, env *Env) (interface{}, error) {
	switch e := expr.(type) {
	case LiteralExpr:
		return e.Value, nil
	case VariableExpr:
		if v, ok := env.Get(e.Name); ok {
			return v, nil
		}
		return nil, fmt.Errorf("variabel %q tidak ditemukan", e.Name)
	case ArrayExpr:
		out := make([]interface{}, len(e.Items))
		for i, x := range e.Items {
			v, err := eval(x, env)
			if err != nil {
				return nil, err
			}
			out[i] = v
		}
		return out, nil
	case ObjectExpr:
		out := make(map[string]interface{}, len(e.Entries))
		for _, entry := range e.Entries {
			v, err := eval(entry.Value, env)
			if err != nil {
				return nil, err
			}
			out[entry.Key] = v
		}
		return out, nil
	case IndexExpr:
		obj, err := eval(e.Object, env)
		if err != nil {
			return nil, err
		}
		idx, err := eval(e.Index, env)
		if err != nil {
			return nil, err
		}
		return indexValue(obj, idx), nil
	case UnaryExpr:
		r, err := eval(e.Right, env)
		if err != nil {
			return nil, err
		}
		switch e.Op {
		case TokenMinus:
			return -number(r), nil
		case TokenPlus:
			return number(r), nil
		case TokenBang:
			return !truth(r), nil
		}
		return nil, nil
	case BinaryExpr:
		return evalBinary(e, env)
	case CallExpr:
		return evalCall(e, env)
	case PropertyExpr:
		o, err := eval(e.Object, env)
		if err != nil {
			return nil, err
		}
		return property(o, e.Name), nil
	default:
		return nil, fmt.Errorf("ekspresi tidak didukung")
	}
}
func evalBinary(e BinaryExpr, env *Env) (interface{}, error) {
	l, err := eval(e.Left, env)
	if err != nil {
		return nil, err
	}
	if e.Op == TokenAndAnd && !truth(l) {
		return false, nil
	}
	if e.Op == TokenOrOr && truth(l) {
		return true, nil
	}
	r, err := eval(e.Right, env)
	if err != nil {
		return nil, err
	}
	switch e.Op {
	case TokenPlus:
		if _, ok := l.(string); ok {
			return formatValue(l) + formatValue(r), nil
		}
		if _, ok := r.(string); ok {
			return formatValue(l) + formatValue(r), nil
		}
		return number(l) + number(r), nil
	case TokenMinus:
		return number(l) - number(r), nil
	case TokenStar:
		return number(l) * number(r), nil
	case TokenSlash:
		if number(r) == 0 {
			return nil, fmt.Errorf("pembagian dengan nol")
		}
		return number(l) / number(r), nil
	case TokenPercent:
		return math.Mod(number(l), number(r)), nil
	case TokenEqualEqual:
		return formatValue(l) == formatValue(r), nil
	case TokenBangEqual:
		return formatValue(l) != formatValue(r), nil
	case TokenLess:
		return number(l) < number(r), nil
	case TokenLessEqual:
		return number(l) <= number(r), nil
	case TokenGreater:
		return number(l) > number(r), nil
	case TokenGreaterEqual:
		return number(l) >= number(r), nil
	case TokenAndAnd:
		return truth(l) && truth(r), nil
	case TokenOrOr:
		return truth(l) || truth(r), nil
	}
	return nil, nil
}
func evalCall(e CallExpr, env *Env) (interface{}, error) {
	callee, err := eval(e.Callee, env)
	if err != nil {
		return nil, err
	}
	args := make([]interface{}, len(e.Args))
	for i, a := range e.Args {
		v, evalErr := eval(a, env)
		if evalErr != nil {
			return nil, evalErr
		}
		args[i] = v
	}
	switch fn := callee.(type) {
	case func([]interface{}) (interface{}, error):
		return fn(args)
	case *Function:
		return callFunction(fn, args)
	default:
		return nil, fmt.Errorf("nilai yang dipanggil bukan fungsi")
	}
}

func callFunction(fn *Function, args []interface{}) (ret interface{}, err error) {
	child := NewEnv(fn.Closure)
	for i, p := range fn.Params {
		if i < len(args) {
			child.Define(p, args[i])
		} else {
			child.Define(p, nil)
		}
	}
	defer func() {
		if r := recover(); r != nil {
			if rs, ok := r.(ReturnSignal); ok {
				ret = rs.Value
				err = nil
				return
			}
			panic(r)
		}
	}()
	err = execBlock(fn.Body, child)
	return nil, err
}
func property(v interface{}, name string) interface{} {
	switch x := v.(type) {
	case map[string]interface{}:
		if value, ok := x[name]; ok {
			return value
		}
		if strings.EqualFold(name, "length") {
			return float64(len(x))
		}
	case []interface{}:
		if strings.EqualFold(name, "length") {
			return float64(len(x))
		}
	case string:
		if strings.EqualFold(name, "length") {
			return float64(len([]rune(x)))
		}
	case Media:
		if name == "path" {
			return x.Path
		}
		if name == "type" || name == "kind" {
			return x.Kind
		}
		if name == "size" {
			return float64(x.Size)
		}
	case *TimelineValue:
		return timelineMethod(x, name)
	case *Font:
		switch strings.ToLower(name) {
		case "path":
			return x.Path
		case "size":
			return float64(x.Size)
		}
	}
	return nil
}

func indexValue(v, index interface{}) interface{} {
	switch x := v.(type) {
	case []interface{}:
		i := int(number(index))
		if i >= 0 && i < len(x) {
			return x[i]
		}
	case map[string]interface{}:
		return x[formatValue(index)]
	case string:
		r := []rune(x)
		i := int(number(index))
		if i >= 0 && i < len(r) {
			return string(r[i])
		}
	}
	return nil
}

func registerBuiltins(env *Env) {
	env.Define("clamp", func(a []interface{}) (interface{}, error) {
		if len(a) < 3 {
			return nil, fmt.Errorf("clamp(value, min, max) membutuhkan 3 argumen")
		}
		v, lo, hi := number(a[0]), number(a[1]), number(a[2])
		if v < lo {
			return lo, nil
		}
		if v > hi {
			return hi, nil
		}
		return v, nil
	})
	env.Define("lerp", func(a []interface{}) (interface{}, error) {
		if len(a) < 3 {
			return nil, fmt.Errorf("lerp(a, b, t) membutuhkan 3 argumen")
		}
		return number(a[0]) + (number(a[1])-number(a[0]))*number(a[2]), nil
	})
	env.Define("easeIn", func(a []interface{}) (interface{}, error) {
		if len(a) < 1 {
			return nil, fmt.Errorf("easeIn(t) membutuhkan 1 argumen")
		}
		t := clamp01(number(a[0]))
		return t * t, nil
	})
	env.Define("easeOut", func(a []interface{}) (interface{}, error) {
		if len(a) < 1 {
			return nil, fmt.Errorf("easeOut(t) membutuhkan 1 argumen")
		}
		t := clamp01(number(a[0]))
		return 1 - (1-t)*(1-t), nil
	})
	env.Define("easeInOut", func(a []interface{}) (interface{}, error) {
		if len(a) < 1 {
			return nil, fmt.Errorf("easeInOut(t) membutuhkan 1 argumen")
		}
		t := clamp01(number(a[0]))
		if t < 0.5 {
			return 2 * t * t, nil
		}
		return 1 - math.Pow(-2*t+2, 2)/2, nil
	})
	env.Define("timeline", func(a []interface{}) (interface{}, error) {
		d := 0.0
		if len(a) > 0 {
			d = number(a[0])
		}
		return NewTimeline(d), nil
	})
	env.Define("keyframe", func(a []interface{}) (interface{}, error) {
		if len(a) < 4 {
			return nil, fmt.Errorf("keyframe(timeline, track, time, value) membutuhkan 4 argumen")
		}
		t, ok := a[0].(*TimelineValue)
		if !ok {
			return nil, fmt.Errorf("argumen pertama keyframe harus timeline")
		}
		t.Add(formatValue(a[1]), number(a[2]), a[3])
		return t, nil
	})
	env.Define("font", func(a []interface{}) (interface{}, error) {
		if len(a) < 1 {
			return nil, fmt.Errorf("font(path, size?) membutuhkan path")
		}
		path := formatValue(a[0])
		size := 48
		if len(a) > 1 {
			size = int(number(a[1]))
		}
		if size < 1 {
			size = 1
		}
		f, err := NewFont(path, size)
		if err != nil {
			return nil, err
		}
		return f, nil
	})
	env.Define("audio", loadMedia)
	env.Define("env", func(a []interface{}) (interface{}, error) {
		if len(a) < 1 {
			return nil, fmt.Errorf("env(name, fallback?) membutuhkan nama")
		}
		name := formatValue(a[0])
		value, ok := os.LookupEnv(name)
		if ok {
			return value, nil
		}
		if len(a) > 1 {
			return formatValue(a[1]), nil
		}
		return "", nil
	})
	env.Define("muxAudio", func(a []interface{}) (interface{}, error) {
		if len(a) < 3 {
			return nil, fmt.Errorf("muxAudio(video, audio, output) membutuhkan 3 argumen")
		}
		return nil, muxAudio(mediaPath(a[0]), mediaPath(a[1]), formatValue(a[2]))
	})
	env.Define("print", func(a []interface{}) (interface{}, error) {
		parts := make([]string, len(a))
		for i, v := range a {
			parts[i] = formatValue(v)
		}
		fmt.Println(strings.Join(parts, " "))
		return nil, nil
	})
	env.Define("sin", unaryNumberBuiltin(math.Sin))
	env.Define("cos", unaryNumberBuiltin(math.Cos))
	env.Define("tan", unaryNumberBuiltin(math.Tan))
	env.Define("abs", unaryNumberBuiltin(math.Abs))
	env.Define("floor", unaryNumberBuiltin(math.Floor))
	env.Define("ceil", unaryNumberBuiltin(math.Ceil))
	env.Define("sqrt", unaryNumberBuiltin(math.Sqrt))
	env.Define("min", func(a []interface{}) (interface{}, error) {
		if len(a) == 0 {
			return nil, fmt.Errorf("min() membutuhkan minimal 1 argumen")
		}
		m := number(a[0])
		for _, v := range a[1:] {
			if number(v) < m {
				m = number(v)
			}
		}
		return m, nil
	})
	env.Define("max", func(a []interface{}) (interface{}, error) {
		if len(a) == 0 {
			return nil, fmt.Errorf("max() membutuhkan minimal 1 argumen")
		}
		m := number(a[0])
		for _, v := range a[1:] {
			if number(v) > m {
				m = number(v)
			}
		}
		return m, nil
	})
	env.Define("readFile", func(a []interface{}) (interface{}, error) {
		if len(a) < 1 {
			return nil, fmt.Errorf("readFile(path) membutuhkan path")
		}
		b, e := os.ReadFile(formatValue(a[0]))
		if e != nil {
			return nil, e
		}
		return string(b), nil
	})
	env.Define("writeFile", func(a []interface{}) (interface{}, error) {
		if len(a) < 2 {
			return nil, fmt.Errorf("writeFile(path, data) membutuhkan 2 argumen")
		}
		p := formatValue(a[0])
		dir := filepath.Dir(p)
		if dir != "." {
			if e := os.MkdirAll(dir, 0755); e != nil {
				return nil, e
			}
		}
		return nil, os.WriteFile(p, []byte(formatValue(a[1])), 0644)
	})
	env.Define("exists", func(a []interface{}) (interface{}, error) {
		if len(a) < 1 {
			return false, fmt.Errorf("exists(path) membutuhkan path")
		}
		_, e := os.Stat(formatValue(a[0]))
		return e == nil, nil
	})
	env.Define("isFile", func(a []interface{}) (interface{}, error) {
		if len(a) < 1 {
			return false, fmt.Errorf("isFile(path) membutuhkan path")
		}
		st, e := os.Stat(formatValue(a[0]))
		if e != nil {
			return false, nil
		}
		return !st.IsDir(), nil
	})
	env.Define("isDir", func(a []interface{}) (interface{}, error) {
		if len(a) < 1 {
			return false, fmt.Errorf("isDir(path) membutuhkan path")
		}
		st, e := os.Stat(formatValue(a[0]))
		if e != nil {
			return false, nil
		}
		return st.IsDir(), nil
	})
	env.Define("fileSize", func(a []interface{}) (interface{}, error) {
		if len(a) < 1 {
			return nil, fmt.Errorf("fileSize(path) membutuhkan path")
		}
		st, e := os.Stat(formatValue(a[0]))
		if e != nil {
			return nil, e
		}
		return float64(st.Size()), nil
	})
	env.Define("mkdir", func(a []interface{}) (interface{}, error) {
		if len(a) < 1 {
			return nil, fmt.Errorf("mkdir(path) membutuhkan path")
		}
		return nil, os.MkdirAll(formatValue(a[0]), 0755)
	})
	env.Define("remove", func(a []interface{}) (interface{}, error) {
		if len(a) < 1 {
			return nil, fmt.Errorf("remove(path) membutuhkan path")
		}
		return nil, os.RemoveAll(formatValue(a[0]))
	})
	env.Define("copyFile", func(a []interface{}) (interface{}, error) {
		if len(a) < 2 {
			return nil, fmt.Errorf("copyFile(src, dst) membutuhkan 2 argumen")
		}
		src, dst := formatValue(a[0]), formatValue(a[1])
		b, e := os.ReadFile(src)
		if e != nil {
			return nil, e
		}
		dir := filepath.Dir(dst)
		if dir != "." {
			if e = os.MkdirAll(dir, 0755); e != nil {
				return nil, e
			}
		}
		return nil, os.WriteFile(dst, b, 0644)
	})
	env.Define("moveFile", func(a []interface{}) (interface{}, error) {
		if len(a) < 2 {
			return nil, fmt.Errorf("moveFile(src, dst) membutuhkan 2 argumen")
		}
		dir := filepath.Dir(formatValue(a[1]))
		if dir != "." {
			if e := os.MkdirAll(dir, 0755); e != nil {
				return nil, e
			}
		}
		return nil, os.Rename(formatValue(a[0]), formatValue(a[1]))
	})
	env.Define("listDir", func(a []interface{}) (interface{}, error) {
		path := "."
		if len(a) > 0 {
			path = formatValue(a[0])
		}
		entries, e := os.ReadDir(path)
		if e != nil {
			return nil, e
		}
		out := make([]interface{}, len(entries))
		for i, entry := range entries {
			out[i] = map[string]interface{}{"name": entry.Name(), "dir": entry.IsDir()}
		}
		return out, nil
	})
	env.Define("pathJoin", func(a []interface{}) (interface{}, error) {
		parts := make([]string, len(a))
		for i, v := range a {
			parts[i] = formatValue(v)
		}
		if len(parts) == 0 {
			return ".", nil
		}
		return filepath.Join(parts...), nil
	})
	env.Define("json", func(a []interface{}) (interface{}, error) {
		if len(a) < 1 {
			return nil, fmt.Errorf("json(value) membutuhkan value")
		}
		b, e := json.Marshal(a[0])
		if e != nil {
			return nil, e
		}
		return string(b), nil
	})
	env.Define("HttpReq", httpReq)
	env.Define("getMedia", getMedia)
	env.Define("loadMedia", loadMedia)
	env.Define("now", func(a []interface{}) (interface{}, error) { return time.Now().Unix(), nil })
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func unaryNumberBuiltin(fn func(float64) float64) func([]interface{}) (interface{}, error) {
	return func(a []interface{}) (interface{}, error) {
		if len(a) < 1 {
			return nil, fmt.Errorf("fungsi membutuhkan 1 argumen")
		}
		return fn(number(a[0])), nil
	}
}

func httpReq(a []interface{}) (interface{}, error) {
	if len(a) < 2 {
		return nil, fmt.Errorf("HttpReq(method, url, body?, headers?) membutuhkan minimal 2 argumen")
	}
	method := strings.ToUpper(formatValue(a[0]))
	url := formatValue(a[1])
	var bodyBytes []byte
	contentType := ""
	if len(a) > 2 && a[2] != nil {
		switch x := a[2].(type) {
		case Media:
			b, e := os.ReadFile(x.Path)
			if e != nil {
				return nil, e
			}
			bodyBytes, contentType = b, mimeForMedia(x)
		case map[string]interface{}, []interface{}:
			b, e := json.Marshal(a[2])
			if e != nil {
				return nil, e
			}
			bodyBytes, contentType = b, "application/json"
		default:
			bodyBytes = []byte(formatValue(a[2]))
		}
	}
	req, e := http.NewRequest(method, url, bytes.NewReader(bodyBytes))
	if e != nil {
		return nil, e
	}
	req.Header.Set("User-Agent", "vscript/0.1")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if len(a) > 3 {
		if headers, ok := a[3].(map[string]interface{}); ok {
			for key, value := range headers {
				req.Header.Set(key, formatValue(value))
			}
		}
	}
	if bodyBytes != nil && req.Header.Get("Content-Type") == "" && method != "GET" && method != "HEAD" {
		req.Header.Set("Content-Type", "application/octet-stream")
	}
	res, e := (&http.Client{Timeout: 60 * time.Second}).Do(req)
	if e != nil {
		return nil, e
	}
	defer res.Body.Close()
	buf := new(bytes.Buffer)
	if _, e = buf.ReadFrom(res.Body); e != nil {
		return nil, e
	}
	headers := map[string]interface{}{}
	for k, values := range res.Header {
		if len(values) > 0 {
			headers[strings.ToLower(k)] = values[0]
		}
	}
	return map[string]interface{}{
		"status":  float64(res.StatusCode),
		"body":    buf.String(),
		"ok":      res.StatusCode >= 200 && res.StatusCode < 300,
		"headers": headers,
	}, nil
}

func mimeForMedia(m Media) string {
	ext := strings.ToLower(filepath.Ext(m.Path))
	switch ext {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	case ".mp3":
		return "audio/mpeg"
	case ".wav":
		return "audio/wav"
	case ".ogg":
		return "audio/ogg"
	case ".mp4":
		return "video/mp4"
	case ".webm":
		return "video/webm"
	default:
		return "application/octet-stream"
	}
}

func getMedia(a []interface{}) (interface{}, error) {
	if len(a) < 1 {
		return nil, fmt.Errorf("getMedia(url) membutuhkan url")
	}
	url := formatValue(a[0])
	req, e := http.NewRequest("GET", url, nil)
	if e != nil {
		return nil, e
	}
	req.Header.Set("User-Agent", "vscript/0.1")
	client := &http.Client{Timeout: 120 * time.Second}
	res, e := client.Do(req)
	if e != nil {
		return nil, e
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("getMedia: http %d", res.StatusCode)
	}
	ct := strings.ToLower(res.Header.Get("Content-Type"))
	if strings.Contains(ct, ";") {
		ct = strings.SplitN(ct, ";", 2)[0]
	}
	ext := filepath.Ext(url)
	if ext == "" {
		switch ct {
		case "image/png":
			ext = ".png"
		case "image/jpeg":
			ext = ".jpg"
		case "image/webp":
			ext = ".webp"
		case "image/gif":
			ext = ".gif"
		case "audio/mpeg":
			ext = ".mp3"
		case "audio/wav", "audio/x-wav":
			ext = ".wav"
		case "audio/ogg":
			ext = ".ogg"
		case "video/mp4":
			ext = ".mp4"
		case "video/webm":
			ext = ".webm"
		default:
			ext = ".bin"
		}
	}
	kind := mediaKind(ct, ext)
	tmp, e := os.CreateTemp("", "vscript-media-*-"+ext)
	if e != nil {
		return nil, e
	}
	name := tmp.Name()
	written, copyErr := io.Copy(tmp, res.Body)
	closeErr := tmp.Close()
	if copyErr != nil {
		os.Remove(name)
		return nil, copyErr
	}
	if closeErr != nil {
		os.Remove(name)
		return nil, closeErr
	}
	return Media{Path: name, Kind: kind, Size: written}, nil
}

func mediaKind(contentType, ext string) string {
	if strings.HasPrefix(contentType, "image/") {
		return "image"
	}
	if strings.HasPrefix(contentType, "audio/") {
		return "audio"
	}
	if strings.HasPrefix(contentType, "video/") {
		return "video"
	}
	ext = strings.ToLower(ext)
	if strings.Contains(".png.jpg.jpeg.webp.bmp.gif", ext) {
		return "image"
	}
	if strings.Contains(".mp3.wav.ogg.flac.aac.m4a", ext) {
		return "audio"
	}
	if strings.Contains(".mp4.mkv.webm.mov.avi", ext) {
		return "video"
	}
	return "binary"
}

func loadMedia(a []interface{}) (interface{}, error) {
	if len(a) < 1 {
		return nil, fmt.Errorf("loadMedia(path) membutuhkan path")
	}
	p := formatValue(a[0])
	st, e := os.Stat(p)
	if e != nil {
		return nil, e
	}
	return Media{Path: p, Kind: mediaKind("", filepath.Ext(p)), Size: st.Size()}, nil
}

func parseHexColor(s string) color.RGBA {
	s = strings.TrimSpace(strings.TrimPrefix(s, "#"))
	if len(s) == 6 {
		var r, g, b uint8
		fmt.Sscanf(s[0:2], "%02x", &r)
		fmt.Sscanf(s[2:4], "%02x", &g)
		fmt.Sscanf(s[4:6], "%02x", &b)
		return color.RGBA{r, g, b, 255}
	}
	return color.RGBA{255, 255, 255, 255}
}

func decodeImage(path string) (image.Image, error) {
	f, e := os.Open(path)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	img, fmtName, e := image.Decode(f)
	_ = fmtName
	return img, e
}
func encodePNG(img image.Image, path string) error {
	f, e := os.Create(path)
	if e != nil {
		return e
	}
	defer f.Close()
	return png.Encode(f, img)
}
func encodeJPEG(img image.Image, path string) error {
	f, e := os.Create(path)
	if e != nil {
		return e
	}
	defer f.Close()
	return jpeg.Encode(f, img, &jpeg.Options{Quality: 92})
}
