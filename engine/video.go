package engine

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type VideoPlan struct {
	Width, Height, FPS int
	Duration           float64
	DesignWidth        int
	DesignHeight       int
	Body               *BlockStmt
	Env                *Env
}

type frameState struct {
	img      *image.RGBA
	renderer *Renderer
	designW  int
	designH  int
}

func (v *VideoPlan) Render(path, codec string, bitrate int64) (err error) {
	if v.Width <= 0 || v.Height <= 0 || v.FPS <= 0 || v.Duration <= 0 {
		return fmt.Errorf("parameter video tidak valid")
	}
	if v.Width%2 != 0 || v.Height%2 != 0 {
		return fmt.Errorf("lebar dan tinggi video harus genap")
	}
	if v.DesignWidth <= 0 || v.DesignHeight <= 0 {
		return fmt.Errorf("design resolution harus positif")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil && filepath.Dir(path) != "." {
		return err
	}
	enc, err := NewFFmpegEncoder(path, v.Width, v.Height, v.FPS, codec, bitrate)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := enc.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
	}()
	r := &Renderer{W: v.Width, H: v.Height, FPS: v.FPS, imageCache: map[string]image.Image{}}
	count := int(math.Ceil(v.Duration * float64(v.FPS)))
	img := image.NewRGBA(image.Rect(0, 0, v.Width, v.Height))
	for frame := 0; frame < count; frame++ {
		// Reuse the framebuffer instead of allocating a new RGBA image every frame.
		clear(img.Pix)
		fs := &frameState{img: img, renderer: r, designW: v.DesignWidth, designH: v.DesignHeight}
		frameTime := float64(frame) / float64(v.FPS)
		// expose frame variables to functions defined outside the video block
		v.Env.Define("t", frameTime)
		v.Env.Define("frame", float64(frame))
		v.Env.Define("fps", float64(v.FPS))
		v.Env.Define("width", float64(v.Width))
		v.Env.Define("height", float64(v.Height))
		local := NewEnv(v.Env)
		local.Define("t", frameTime)
		local.Define("frame", float64(frame))
		local.Define("fps", float64(v.FPS))
		local.Define("width", float64(v.Width))
		local.Define("height", float64(v.Height))
		local.Define("_frame", fs)
		if err := execVideoBlock(v.Body, local); err != nil {
			return fmt.Errorf("frame %d: %w", frame, err)
		}
		if err := enc.WriteRGBA(img.Pix, int64(frame)); err != nil {
			return err
		}
	}
	return nil
}

func execVideoBlock(b *BlockStmt, env *Env) error {
	for _, s := range b.Statements {
		if err := execVideoStmt(s, env); err != nil {
			return err
		}
	}
	return nil
}

func execVideoStmt(stmt Stmt, env *Env) error {
	switch s := stmt.(type) {
	case ExprStmt:
		if call, ok := s.Expr.(CallExpr); ok {
			return videoCall(call, env)
		}
		_, err := eval(s.Expr, env)
		return err
	case LetStmt:
		v, e := eval(s.Value, env)
		if e != nil {
			return e
		}
		env.Define(s.Name, v)
		return nil
	case AssignStmt:
		v, e := eval(s.Value, env)
		if e != nil {
			return e
		}
		if !env.Set(s.Name, v) {
			env.Define(s.Name, v)
		}
		return nil
	case IfStmt:
		c, e := eval(s.Cond, env)
		if e != nil {
			return e
		}
		if truth(c) {
			return execVideoBlock(s.Then, env)
		}
		if s.Else != nil {
			return execVideoBlock(s.Else, env)
		}
		return nil
	case ForStmt:
		startValue, e := eval(s.Start, env)
		if e != nil {
			return e
		}
		endValue, e := eval(s.End, env)
		if e != nil {
			return e
		}
		start := int(number(startValue))
		end := int(number(endValue))
		step := 1
		if start > end {
			step = -1
		}
		for i := start; ; i += step {
			env.Define(s.Name, float64(i))
			control, loopErr := executeVideoLoopBody(s.Body, env)
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
			cond, e := eval(s.Cond, env)
			if e != nil {
				return e
			}
			if !truth(cond) {
				break
			}
			control, loopErr := executeVideoLoopBody(s.Body, env)
			if loopErr != nil {
				return loopErr
			}
			if control == 2 {
				break
			}
		}
		return nil
	case FunctionStmt:
		env.Define(s.Name, &Function{Params: s.Params, Body: s.Body, Closure: env})
		return nil
	case BreakStmt:
		panic(BreakSignal{})
	case ContinueStmt:
		panic(ContinueSignal{})
	default:
		return execStmt(stmt, env)
	}
}

func executeVideoLoopBody(body *BlockStmt, env *Env) (control int, err error) {
	defer func() {
		if r := recover(); r != nil {
			switch r.(type) {
			case BreakSignal:
				control, err = 2, nil
			case ContinueSignal:
				control, err = 1, nil
			default:
				panic(r)
			}
		}
	}()
	if e := execVideoBlock(body, env); e != nil {
		return 0, e
	}
	return 0, nil
}

func videoCall(c CallExpr, env *Env) error {
	name := ""
	if v, ok := c.Callee.(VariableExpr); ok {
		name = v.Name
	}
	fs, ok := env.Get("_frame")
	if !ok {
		return nil
	}
	state := fs.(*frameState)
	args := make([]interface{}, len(c.Args))
	for i, a := range c.Args {
		v, e := eval(a, env)
		if e != nil {
			return e
		}
		args[i] = v
	}
	if state.designW <= 0 {
		state.designW = state.img.Bounds().Dx()
	}
	if state.designH <= 0 {
		state.designH = state.img.Bounds().Dy()
	}
	switch name {
	case "background":
		if len(args) < 1 {
			return fmt.Errorf("background membutuhkan color")
		}
		c, err := parseHexColor(formatValue(args[0]))
		if err != nil {
			return err
		}
		return fill(state.img, c)
	case "rect":
		if len(args) < 5 {
			return fmt.Errorf("rect membutuhkan x, y, width, height, color")
		}
		c, err := parseHexColor(formatValue(args[4]))
		if err != nil {
			return err
		}
		return drawRect(state.img, vX(state, args[0]), vY(state, args[1]), vX(state, args[2]), vY(state, args[3]), c, vRadius(state, optionalRadius(args, 5)))
	case "circle":
		if len(args) < 4 {
			return fmt.Errorf("circle membutuhkan x, y, radius, color")
		}
		c, err := parseHexColor(formatValue(args[3]))
		if err != nil {
			return err
		}
		return drawCircle(state.img, vX(state, args[0]), vY(state, args[1]), vRadius(state, args[2]), c)
	case "line":
		if len(args) < 6 {
			return fmt.Errorf("line membutuhkan 6 argumen")
		}
		c, err := parseHexColor(formatValue(args[4]))
		if err != nil {
			return err
		}
		return drawLine(state.img, vX(state, args[0]), vY(state, args[1]), vX(state, args[2]), vY(state, args[3]), c, vRadius(state, args[5]))
	case "polygon":
		if len(args) < 2 {
			return fmt.Errorf("polygon membutuhkan points dan color")
		}
		c, err := parseHexColor(formatValue(args[1]))
		if err != nil {
			return err
		}
		return drawPolygon(state.img, scalePoints(args[0], state), c)
	case "text":
		if len(args) < 5 {
			return fmt.Errorf("text membutuhkan text, x, y, size, color")
		}
		px, py, size := vX(state, args[1]), vY(state, args[2]), vRadius(state, args[3])
		c, err := parseHexColor(formatValue(args[4]))
		if err != nil {
			return err
		}
		if len(args) > 5 {
			font, ok := optionalFont(args, 5)
			if !ok {
				return fmt.Errorf("argumen font pada text harus berupa object font")
			}
			return font.DrawRGBA(state.img.Pix, state.img.Bounds().Dx(), state.img.Bounds().Dy(), state.img.Stride, px, py, size, rgbaArray(c), formatValue(args[0]))
		}
		return drawText(state.img, formatValue(args[0]), px, py, size, c)
	case "image":
		if len(args) < 5 {
			return fmt.Errorf("image membutuhkan x, y, width, height, media")
		}
		return drawImage(state.img, imgMediaPath(args[4]), vX(state, args[0]), vY(state, args[1]), vX(state, args[2]), vY(state, args[3]), state.renderer.imageCache)
	case "filter":
		if len(args) < 1 {
			return fmt.Errorf("filter membutuhkan nama")
		}
		filterName := formatValue(args[0])
		if fnValue, ok := env.Get(filterName); ok {
			if fn, ok := fnValue.(*Function); ok {
				return applyScriptFilter(state.img, fn)
			}
		}
		return applyFilter(state.img, filterName, args[1:])
	case "crop":
		if len(args) < 4 {
			return fmt.Errorf("crop membutuhkan x, y, width, height")
		}
		return applyCrop(state.img, vX(state, args[0]), vY(state, args[1]), vX(state, args[2]), vY(state, args[3]))
	case "cropAspect":
		if len(args) < 1 {
			return fmt.Errorf("cropAspect membutuhkan rasio seperti \"16:9\"")
		}
		return applyCropAspect(state.img, formatValue(args[0]))
	case "pixelate":
		if len(args) < 1 {
			return fmt.Errorf("pixelate membutuhkan ukuran")
		}
		return applyPixelate(state.img, vRadius(state, args[0]))
	case "saveFrame":
		if len(args) < 1 {
			return fmt.Errorf("saveFrame membutuhkan path")
		}
		return saveFrame(state.img, formatValue(args[0]))
	}
	return fmt.Errorf("perintah video %q tidak dikenal", name)
}

func rgbaArray(c color.RGBA) [4]uint8 { return [4]uint8{c.R, c.G, c.B, c.A} }

func optionalFont(a []interface{}, i int) (*Font, bool) {
	if len(a) <= i {
		return nil, false
	}
	f, ok := a[i].(*Font)
	return f, ok && f != nil
}

func vX(s *frameState, v interface{}) int {
	return int(math.Round(number(v) * float64(s.img.Bounds().Dx()) / float64(s.designW)))
}
func vY(s *frameState, v interface{}) int {
	return int(math.Round(number(v) * float64(s.img.Bounds().Dy()) / float64(s.designH)))
}
func vRadius(s *frameState, v interface{}) int {
	scale := (float64(s.img.Bounds().Dx())/float64(s.designW) + float64(s.img.Bounds().Dy())/float64(s.designH)) / 2
	return int(math.Round(number(v) * scale))
}
func scalePoints(v interface{}, s *frameState) interface{} {
	a, ok := v.([]interface{})
	if !ok {
		return v
	}
	out := make([]interface{}, len(a))
	for i, x := range a {
		if i%2 == 0 {
			out[i] = float64(vX(s, x))
		} else {
			out[i] = float64(vY(s, x))
		}
	}
	return out
}

func optionalRadius(a []interface{}, i int) int {
	if len(a) > i {
		return int(number(a[i]))
	}
	return 0
}
func imgMediaPath(v interface{}) string {
	if m, ok := v.(Media); ok {
		return m.Path
	}
	return formatValue(v)
}
func fill(img *image.RGBA, c color.RGBA) error {
	for y := 0; y < img.Bounds().Dy(); y++ {
		row := y * img.Stride
		for x := 0; x < img.Bounds().Dx(); x++ {
			off := row + x*4
			img.Pix[off] = c.R
			img.Pix[off+1] = c.G
			img.Pix[off+2] = c.B
			img.Pix[off+3] = c.A
		}
	}
	return nil
}
func drawRect(img *image.RGBA, x, y, w, h int, c color.RGBA, radius int) error {
	if w <= 0 || h <= 0 {
		return fmt.Errorf("ukuran rect harus positif")
	}
	if radius < 0 {
		radius = 0
	}
	if radius > minInt(w, h)/2 {
		radius = minInt(w, h) / 2
	}
	minx := maxInt(0, x)
	miny := maxInt(0, y)
	maxx := minInt(img.Bounds().Dx(), x+w)
	maxy := minInt(img.Bounds().Dy(), y+h)
	if minx >= maxx || miny >= maxy {
		return nil
	}
	for yy := miny; yy < maxy; yy++ {
		for xx := minx; xx < maxx; xx++ {
			if radius > 0 {
				dx := float64(xx - x)
				dy := float64(yy - y)
				r := float64(radius)
				inside := true
				switch {
				case dx < r && dy < r:
					qx, qy := dx-r, dy-r
					inside = qx*qx+qy*qy <= r*r
				case dx >= float64(w)-r && dy < r:
					qx, qy := dx-(float64(w)-r), dy-r
					inside = qx*qx+qy*qy <= r*r
				case dx < r && dy >= float64(h)-r:
					qx, qy := dx-r, dy-(float64(h)-r)
					inside = qx*qx+qy*qy <= r*r
				case dx >= float64(w)-r && dy >= float64(h)-r:
					qx, qy := dx-(float64(w)-r), dy-(float64(h)-r)
					inside = qx*qx+qy*qy <= r*r
				}
				if !inside {
					continue
				}
			}
			img.SetRGBA(xx, yy, c)
		}
	}
	return nil
}

func drawCircle(img *image.RGBA, cx, cy, r int, c color.RGBA) error {
	if r < 0 {
		return fmt.Errorf("radius circle tidak boleh negatif")
	}
	if r == 0 {
		if cx >= 0 && cy >= 0 && cx < img.Bounds().Dx() && cy < img.Bounds().Dy() {
			img.SetRGBA(cx, cy, c)
		}
		return nil
	}
	rr := int64(r) * int64(r)
	for y := cy - r; y <= cy+r; y++ {
		for x := cx - r; x <= cx+r; x++ {
			if x < 0 || y < 0 || x >= img.Bounds().Dx() || y >= img.Bounds().Dy() {
				continue
			}
			dx := int64(x - cx)
			dy := int64(y - cy)
			if dx*dx+dy*dy <= rr {
				img.SetRGBA(x, y, c)
			}
		}
	}
	return nil
}

func drawLine(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA, width int) error {
	if width < 1 {
		width = 1
	}
	dx := float64(x1 - x0)
	dy := float64(y1 - y0)
	steps := int(math.Max(math.Abs(dx), math.Abs(dy)))
	for i := 0; i <= steps; i++ {
		u := 0.0
		if steps > 0 {
			u = float64(i) / float64(steps)
		}
		x := int(math.Round(float64(x0) + dx*u))
		y := int(math.Round(float64(y0) + dy*u))
		if err := drawCircle(img, x, y, maxInt(1, width/2), c); err != nil {
			return err
		}
	}
	return nil
}

var builtinGlyphs = map[rune][7]string{
	'a': {"01110", "10001", "10001", "11111", "10001", "10001", "10001"},
	'b': {"11110", "10001", "10001", "11110", "10001", "10001", "11110"},
	'c': {"01111", "10000", "10000", "10000", "10000", "10000", "01111"},
	'd': {"11110", "10001", "10001", "10001", "10001", "10001", "11110"},
	'e': {"11111", "10000", "10000", "11110", "10000", "10000", "11111"},
	'f': {"11111", "10000", "10000", "11110", "10000", "10000", "10000"},
	'g': {"01111", "10000", "10000", "10111", "10001", "10001", "01111"},
	'h': {"10001", "10001", "10001", "11111", "10001", "10001", "10001"},
	'i': {"11111", "00100", "00100", "00100", "00100", "00100", "11111"},
	'j': {"00111", "00010", "00010", "00010", "00010", "10010", "01100"},
	'k': {"10001", "10010", "10100", "11000", "10100", "10010", "10001"},
	'l': {"10000", "10000", "10000", "10000", "10000", "10000", "11111"},
	'm': {"10001", "11011", "10101", "10101", "10001", "10001", "10001"},
	'n': {"10001", "11001", "10101", "10011", "10001", "10001", "10001"},
	'o': {"01110", "10001", "10001", "10001", "10001", "10001", "01110"},
	'p': {"11110", "10001", "10001", "11110", "10000", "10000", "10000"},
	'q': {"01110", "10001", "10001", "10001", "10101", "10010", "01101"},
	'r': {"11110", "10001", "10001", "11110", "10100", "10010", "10001"},
	's': {"01111", "10000", "10000", "01110", "00001", "00001", "11110"},
	't': {"11111", "00100", "00100", "00100", "00100", "00100", "00100"},
	'u': {"10001", "10001", "10001", "10001", "10001", "10001", "01110"},
	'v': {"10001", "10001", "10001", "10001", "10001", "01010", "00100"},
	'w': {"10001", "10001", "10001", "10101", "10101", "11011", "10001"},
	'x': {"10001", "10001", "01010", "00100", "01010", "10001", "10001"},
	'y': {"10001", "10001", "01010", "00100", "00100", "00100", "00100"},
	'z': {"11111", "00001", "00010", "00100", "01000", "10000", "11111"},
	' ': {"00000", "00000", "00000", "00000", "00000", "00000", "00000"},
	'0': {"01110", "10001", "10011", "10101", "11001", "10001", "01110"},
	'1': {"00100", "01100", "00100", "00100", "00100", "00100", "01110"},
	'2': {"01110", "10001", "00001", "00010", "00100", "01000", "11111"},
	'3': {"11110", "00001", "00001", "01110", "00001", "00001", "11110"},
	'4': {"00010", "00110", "01010", "10010", "11111", "00010", "00010"},
	'5': {"11111", "10000", "10000", "11110", "00001", "00001", "11110"},
	'6': {"01110", "10000", "10000", "11110", "10001", "10001", "01110"},
	'7': {"11111", "00001", "00010", "00100", "01000", "01000", "01000"},
	'8': {"01110", "10001", "10001", "01110", "10001", "10001", "01110"},
	'9': {"01110", "10001", "10001", "01111", "00001", "00001", "01110"},
	'.': {"00000", "00000", "00000", "00000", "00000", "00110", "00110"},
	',': {"00000", "00000", "00000", "00000", "00000", "00110", "01100"},
	'!': {"00100", "00100", "00100", "00100", "00100", "00000", "00100"},
	'?': {"01110", "10001", "00001", "00010", "00100", "00000", "00100"},
	':': {"00000", "00110", "00110", "00000", "00110", "00110", "00000"},
	';': {"00000", "00110", "00110", "00000", "00110", "00100", "01000"},
	'-': {"00000", "00000", "00000", "01110", "00000", "00000", "00000"},
	'_': {"00000", "00000", "00000", "00000", "00000", "00000", "11111"},
	'/': {"00001", "00010", "00010", "00100", "01000", "01000", "10000"},
	'@': {"01110", "10001", "10111", "10101", "10111", "10000", "01111"},
	'#': {"01010", "11111", "01010", "01010", "11111", "01010", "00000"},
	'&': {"01100", "10010", "10100", "01000", "10101", "10010", "01101"},
	'+': {"00000", "00100", "00100", "11111", "00100", "00100", "00000"},
	'=': {"00000", "11111", "00000", "11111", "00000", "00000", "00000"},
	'%': {"11001", "11010", "00010", "00100", "01000", "01011", "10011"},
	'(': {"00010", "00100", "01000", "01000", "01000", "00100", "00010"},
	')': {"01000", "00100", "00010", "00010", "00010", "00100", "01000"},
}

func drawText(img *image.RGBA, text string, cx, cy, size int, c color.RGBA) error {
	if size < 1 {
		size = 1
	}
	lower := []rune(strings.ToLower(text))
	if len(lower) == 0 {
		return nil
	}
	// The public size is the approximate glyph height, not the scale of each 5x7 cell.
	cell := float64(size) / 7.0
	if cell < 1 {
		cell = 1
	}
	advance := 6 * cell
	total := float64(len(lower)) * advance
	start := float64(cx) - total/2
	top := float64(cy) - float64(size)/2
	for i, ch := range lower {
		glyph, ok := builtinGlyphs[ch]
		if !ok {
			glyph = builtinGlyphs[' ']
		}
		glyphStart := start + float64(i)*advance
		for gy, row := range glyph {
			for gx, b := range row {
				if b != '1' {
					continue
				}
				x0 := int(math.Round(glyphStart + float64(gx)*cell))
				x1 := int(math.Round(glyphStart + float64(gx+1)*cell))
				y0 := int(math.Round(top + float64(gy)*cell))
				y1 := int(math.Round(top + float64(gy+1)*cell))
				if x1 <= x0 {
					x1 = x0 + 1
				}
				if y1 <= y0 {
					y1 = y0 + 1
				}
				for py := y0; py < y1; py++ {
					for px := x0; px < x1; px++ {
						if px >= 0 && py >= 0 && px < img.Bounds().Dx() && py < img.Bounds().Dy() {
							img.SetRGBA(px, py, c)
						}
					}
				}
			}
		}
	}
	return nil
}

func drawPolygon(img *image.RGBA, pointsValue interface{}, c color.RGBA) error {
	points, ok := pointsValue.([]interface{})
	if !ok || len(points) < 6 || len(points)%2 != 0 {
		return fmt.Errorf("polygon points harus array x,y berpasangan")
	}
	type point struct{ x, y int }
	poly := make([]point, 0, len(points)/2)
	minX, minY := img.Bounds().Dx(), img.Bounds().Dy()
	maxX, maxY := 0, 0
	for i := 0; i < len(points); i += 2 {
		x, y := int(number(points[i])), int(number(points[i+1]))
		poly = append(poly, point{x: x, y: y})
		minX = minInt(minX, x)
		minY = minInt(minY, y)
		maxX = maxInt(maxX, x)
		maxY = maxInt(maxY, y)
	}
	minX = maxInt(0, minX)
	minY = maxInt(0, minY)
	maxX = minInt(img.Bounds().Dx()-1, maxX)
	maxY = minInt(img.Bounds().Dy()-1, maxY)
	if minX > maxX || minY > maxY {
		return nil
	}
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			inside := false
			j := len(poly) - 1
			for i := 0; i < len(poly); i++ {
				yi, yj := poly[i].y, poly[j].y
				xi, xj := poly[i].x, poly[j].x
				intersects := (yi > y) != (yj > y) && float64(x) < float64(xj-xi)*float64(y-yi)/float64(yj-yi)+float64(xi)
				if intersects {
					inside = !inside
				}
				j = i
			}
			if inside {
				img.SetRGBA(x, y, c)
			}
		}
	}
	return nil
}

func applyScriptFilter(img *image.RGBA, fn *Function) error {
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			off := y*img.Stride + x*4
			ret, err := callFunction(fn, []interface{}{float64(x), float64(y), float64(img.Pix[off]), float64(img.Pix[off+1]), float64(img.Pix[off+2]), float64(img.Pix[off+3])})
			if err != nil {
				return fmt.Errorf("custom filter: %w", err)
			}
			vals, ok := ret.([]interface{})
			if !ok || len(vals) < 3 || len(vals) > 4 {
				return fmt.Errorf("custom filter harus return [r,g,b] atau [r,g,b,a]")
			}
			img.Pix[off] = clamp8(int(number(vals[0])))
			img.Pix[off+1] = clamp8(int(number(vals[1])))
			img.Pix[off+2] = clamp8(int(number(vals[2])))
			if len(vals) == 4 {
				img.Pix[off+3] = clamp8(int(number(vals[3])))
			}
		}
	}
	return nil
}

func drawImage(img *image.RGBA, path string, x, y, w, h int, cache map[string]image.Image) error {
	if path == "" || w <= 0 || h <= 0 {
		return fmt.Errorf("image membutuhkan path dan ukuran positif")
	}
	var src image.Image
	var e error
	if cache != nil {
		src = cache[path]
	}
	if src == nil {
		src, e = decodeImage(path)
		if e != nil {
			return e
		}
		if cache != nil {
			cache[path] = src
		}
	}
	sb := src.Bounds()
	for yy := 0; yy < h; yy++ {
		py := y + yy
		if py < 0 || py >= img.Bounds().Dy() {
			continue
		}
		sy := sb.Min.Y + yy*sb.Dy()/h
		for xx := 0; xx < w; xx++ {
			px := x + xx
			if px < 0 || px >= img.Bounds().Dx() {
				continue
			}
			sx := sb.Min.X + xx*sb.Dx()/w
			r, g, b, a := src.At(sx, sy).RGBA()
			img.SetRGBA(px, py, color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)})
		}
	}
	return nil
}
func applyFilter(img *image.RGBA, name string, args []interface{}) error {
	filterName := strings.ToLower(name)
	require := func(n int) error {
		if len(args) < n {
			return fmt.Errorf("filter %q membutuhkan %d argumen", filterName, n)
		}
		return nil
	}
	switch filterName {
	case "grayscale":
		for i := 0; i < len(img.Pix); i += 4 {
			v := uint8(float64(img.Pix[i])*0.299 + float64(img.Pix[i+1])*0.587 + float64(img.Pix[i+2])*0.114)
			img.Pix[i], img.Pix[i+1], img.Pix[i+2] = v, v, v
		}
	case "invert":
		for i := 0; i < len(img.Pix); i += 4 {
			img.Pix[i] = 255 - img.Pix[i]
			img.Pix[i+1] = 255 - img.Pix[i+1]
			img.Pix[i+2] = 255 - img.Pix[i+2]
		}
	case "brightness":
		if e := require(1); e != nil {
			return e
		}
		delta := int(number(args[0]))
		for i := 0; i < len(img.Pix); i += 4 {
			img.Pix[i] = clamp8(int(img.Pix[i]) + delta)
			img.Pix[i+1] = clamp8(int(img.Pix[i+1]) + delta)
			img.Pix[i+2] = clamp8(int(img.Pix[i+2]) + delta)
		}
	case "contrast":
		if e := require(1); e != nil {
			return e
		}
		factor := number(args[0])
		for i := 0; i < len(img.Pix); i += 4 {
			img.Pix[i] = clamp8(int((float64(img.Pix[i])-128)*factor + 128))
			img.Pix[i+1] = clamp8(int((float64(img.Pix[i+1])-128)*factor + 128))
			img.Pix[i+2] = clamp8(int((float64(img.Pix[i+2])-128)*factor + 128))
		}
	case "vignette":
		if e := require(1); e != nil {
			return e
		}
		strength := number(args[0])
		cx := float64(img.Bounds().Dx()) / 2
		cy := float64(img.Bounds().Dy()) / 2
		maxd := math.Hypot(cx, cy)
		for y := 0; y < img.Bounds().Dy(); y++ {
			for x := 0; x < img.Bounds().Dx(); x++ {
				d := math.Hypot(float64(x)-cx, float64(y)-cy) / maxd
				mul := 1 - strength*d*d
				off := y*img.Stride + x*4
				img.Pix[off] = clamp8(int(float64(img.Pix[off]) * mul))
				img.Pix[off+1] = clamp8(int(float64(img.Pix[off+1]) * mul))
				img.Pix[off+2] = clamp8(int(float64(img.Pix[off+2]) * mul))
			}
		}
	case "blur":
		if e := require(1); e != nil {
			return e
		}
		radius := int(number(args[0]))
		return blur(img, radius)
	case "sepia":
		for i := 0; i < len(img.Pix); i += 4 {
			r, g, b := float64(img.Pix[i]), float64(img.Pix[i+1]), float64(img.Pix[i+2])
			img.Pix[i] = clamp8(int(r*.393 + g*.769 + b*.189))
			img.Pix[i+1] = clamp8(int(r*.349 + g*.686 + b*.168))
			img.Pix[i+2] = clamp8(int(r*.272 + g*.534 + b*.131))
		}
	case "scanlines":
		if e := require(1); e != nil {
			return e
		}
		amount := number(args[0])
		for y := 0; y < img.Bounds().Dy(); y++ {
			if y%2 == 0 {
				for x := 0; x < img.Bounds().Dx(); x++ {
					off := y*img.Stride + x*4
					mul := 1 - amount
					img.Pix[off] = clamp8(int(float64(img.Pix[off]) * mul))
					img.Pix[off+1] = clamp8(int(float64(img.Pix[off+1]) * mul))
					img.Pix[off+2] = clamp8(int(float64(img.Pix[off+2]) * mul))
				}
			}
		}
	default:
		return fmt.Errorf("filter %q tidak dikenal", name)
	}
	return nil
}
func blur(img *image.RGBA, r int) error {
	if r <= 0 {
		return nil
	}
	tmp := append([]byte(nil), img.Pix...)
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var sr, sg, sb, sa, n int
			for ky := maxInt(0, y-r); ky <= minInt(h-1, y+r); ky++ {
				for kx := maxInt(0, x-r); kx <= minInt(w-1, x+r); kx++ {
					off := ky*img.Stride + kx*4
					sr += int(tmp[off])
					sg += int(tmp[off+1])
					sb += int(tmp[off+2])
					sa += int(tmp[off+3])
					n++
				}
			}
			off := y*img.Stride + x*4
			img.Pix[off] = uint8(sr / n)
			img.Pix[off+1] = uint8(sg / n)
			img.Pix[off+2] = uint8(sb / n)
			img.Pix[off+3] = uint8(sa / n)
		}
	}
	return nil
}
func applyCrop(img *image.RGBA, x, y, w, h int) error {
	if w <= 0 || h <= 0 {
		return nil
	}
	src := append([]byte(nil), img.Pix...)
	ow, oh := img.Bounds().Dx(), img.Bounds().Dy()
	for yy := 0; yy < oh; yy++ {
		sy := y + yy*h/oh
		for xx := 0; xx < ow; xx++ {
			sx := x + xx*w/ow
			off := yy*img.Stride + xx*4
			if sx >= 0 && sy >= 0 && sx < ow && sy < oh {
				so := sy*img.Stride + sx*4
				copy(img.Pix[off:off+4], src[so:so+4])
			}
		}
	}
	return nil
}
func applyCropAspect(img *image.RGBA, ratio string) error {
	parts := strings.Split(strings.TrimSpace(ratio), ":")
	if len(parts) != 2 {
		return fmt.Errorf("rasio crop harus berbentuk W:H")
	}
	rw, rh := numberString(parts[0]), numberString(parts[1])
	if rw <= 0 || rh <= 0 {
		return fmt.Errorf("rasio crop tidak valid")
	}
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	target := rw / rh
	current := float64(w) / float64(h)
	if current > target {
		cropW := int(float64(h) * target)
		x := (w - cropW) / 2
		return applyCrop(img, x, 0, cropW, h)
	}
	cropH := int(float64(w) / target)
	y := (h - cropH) / 2
	return applyCrop(img, 0, y, w, cropH)
}

func numberString(s string) float64 {
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return v
}

func applyPixelate(img *image.RGBA, size int) error {
	if size < 2 {
		return nil
	}
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	for y := 0; y < h; y += size {
		for x := 0; x < w; x += size {
			off := y*img.Stride + x*4
			r, g, b, a := img.Pix[off], img.Pix[off+1], img.Pix[off+2], img.Pix[off+3]
			for yy := y; yy < minInt(h, y+size); yy++ {
				for xx := x; xx < minInt(w, x+size); xx++ {
					o := yy*img.Stride + xx*4
					img.Pix[o], img.Pix[o+1], img.Pix[o+2], img.Pix[o+3] = r, g, b, a
				}
			}
		}
	}
	return nil
}
func saveFrame(img *image.RGBA, path string) error {
	dir := filepath.Dir(path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".jpg" || ext == ".jpeg" {
		return encodeJPEG(img, path)
	}
	return encodePNG(img, path)
}
func clamp8(v int) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
