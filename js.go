package grid

import (
	"syscall/js"
)

// A type representing the javaScript
// object that is passed to NewGrid
// to specify the grid settings.
type GridObj struct {
	id         string
	class      string
	width      int
	height     int
	cellWidth  int
	cellHeight int
	speed      int
}

// Class to style list.
var css = map[string][]string{}

// Store grids for access from javascript.
var grids = map[string]*grid{}

// Apply the css styles stored in css.
func ApplyCss(el *js.Value, class string) {
	list := css[class]
	style := ""
	for i := range list {
		style += list[i] + ";"
	}
	el.Set("style", style)
}

func AddCssStyle(class, style string) {
	if s, ok := css[class]; !ok {
		css[class] = []string{style}
	} else {
		s = append(s, style)
	}
}

// TODO: Css selector rules can be applied to the class string
// to map styles to elements.
func SetCssMap(this js.Value, inputs []js.Value) interface{} {
	for _, obj := range inputs {
		class := obj.Get("class").String()
		styles := obj.Get("styles")
		s := []string{}
		for i := 0; i < styles.Length(); i++ {
			s = append(s, styles.Index(i).String())
		}
		css[class] = s
	}
	return nil
}

// Helper for getting bounding client rect.
func getBounds(cnv js.Value) (int, int) {
	bounds := cnv.Call("getBoundingClientRect")
	x := bounds.Get("left").Int()
	y := bounds.Get("top").Int()
	return x, y
}

// Helper for getting window scroll values.
func getScrollCoords() (int, int) {
	x := js.Global().Get("window").Get("scrollX").Int()
	y := js.Global().Get("window").Get("scrollY").Int()
	return x, y
}

// Helper for creating dom elements.
func CreateElement(typ string) js.Value {
	doc := js.Global().Get("document")
	return doc.Call("createElement", typ)
}

// Create the canvas that will be used as the background.
// TODO: consider making the strokeStyle border color a setting.
func createBackGround(width, height, cellWidth, cellHeight int) js.Value {
	cnv := CreateElement("canvas")
	cnv.Set("width", width*2)
	cnv.Set("height", height*2)
	ctx := cnv.Call("getContext", "2d")
	ctx.Set("fillStyle", "white")
	ctx.Call("fillRect", 0, 0, width*2, height*2)
	ctx.Set("lineWidth", 0.25)
	ctx.Call("beginPath")
	rows := height*2/cellHeight + 1
	cols := width*2/cellWidth + 1
	for i := 0; i < cols; i++ {
		ctx.Call("moveTo", i*cellWidth, 0)
		ctx.Call("lineTo", i*cellWidth, height*2)
	}
	for i := 0; i < rows; i++ {
		ctx.Call("moveTo", 0, i*cellHeight)
		ctx.Call("lineTo", width*2, i*cellHeight)
	}
	ctx.Call("stroke")
	return cnv
}

// Create the view-port canvas to draw the foreground of the grid.
func createView(width, height int, main js.Value) (js.Value, js.Value) {
	cnv := CreateElement("canvas")
	ctx := cnv.Call("getContext", "2d")
	cnv.Set("width", width)
	cnv.Set("height", height)
	main.Call("appendChild", cnv)
	return ctx, cnv
}

// Create a new grid object from json.
// Note that if the width and height are not divisible by the
// cellWidth and cellHeight they will be adjusted.
func NewGridObj(obj js.Value) GridObj {
	g := GridObj{}
	g.id = obj.Get("id").String()
	g.class = obj.Get("class").String()
	g.width = obj.Get("width").Int()
	g.height = obj.Get("height").Int()
	g.cellWidth = obj.Get("cellWidth").Int()
	g.cellHeight = obj.Get("cellHeight").Int()

	// Normalize grid width and height to cell width and height.
	// This makes the math for the background recycle simpler.
	g.width = g.width / g.cellWidth * g.cellWidth
	g.height = g.height / g.cellHeight * g.cellHeight

	g.speed = obj.Get("scroll-speed").Int()
	return g
}

// External JavaScript function to add data to a grid.
// args: "grid id", row, col.
func AddData(this js.Value, args[]js.Value) interface{} {
	id := args[0].String()
	row := args[1].Int()
	col := args[2].Int()
	value := args[3].String()
	g := grids[id]
	g.addData(row, col, value)
	g.Draw()
	return nil
}

// External JavaScript function to create a new grid.
// args: JSON object(GridObj).
func NewGridJs(this js.Value, args[]js.Value) interface{} {
	obj := NewGridObj(args[0])
	g := NewGrid(obj)
	g.Draw()
	return g.GetElement()
}
