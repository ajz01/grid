package grid

import (
	"fmt"
	"syscall/js"
)

// A cell of the grid.
type Cell struct {
	X           int
	Y           int
	Row         int
	Col         int
	Value       string
	Editing     bool
	Grid	*grid
}

// The address of a cell.
type Address struct {
	Row int
	Col int
}

// The main grid type.
type grid struct {
	class          string
	x, y, sx, sy   int
	width, height  int
	vcnv, cnv, ctx js.Value
	main           js.Value
	selectedCells  map[Address]*Cell
	data           map[Address]*Cell
	cellWidth      int
	cellHeight     int
	direction      int
	interval       js.Value
	speed          int
	editCell       *Cell
	scrolling      bool
	active         bool
	mouseDown      bool
	scrollAmt	int
	lastScroll	int
	container Container
}

// The public interface for a grid.
type Grid interface {
	Draw()
	AddEventHandler(event string, handler func(this js.Value, args []js.Value) interface{})
	GetCtx() *js.Value
	GetCellWidth() int
	GetCellHeight() int
	GetX() int
	GetY() int
	GetEditCellAddress() *Address
	AddContainer(container Container)
	GetElement() js.Value
	AddData(row, col int, value string)
}

// The Container interface provides the methods for the grid.container.
// The grid.container allows for adding event handlers and cell styles
// to the grid while letting the grid handle the standard events and
// cell styles.
type Container interface {
	AddCell(cell *Cell)
	SetCellStyles(row, col int)
	SetCellFontStyles(row, col int)
}

func (g *grid) Draw() {
	g.draw()
}

func (g *grid) AddData(row, col int, value string) {
	g.addData(row, col, value)
}

func (g *grid) AddEventHandler(event string, handler func(this js.Value, args []js.Value) interface{}) {
	g.vcnv.Call("addEventListener", event, js.FuncOf(handler))
}

func (g grid) GetCtx() *js.Value {
	return &g.ctx
}

func (g grid) GetCellWidth() int {
	return g.cellWidth
}

func (g grid) GetCellHeight() int {
	return g.cellHeight
}

func (g grid) GetX() int {
	return g.x
}

func (g grid) GetY() int {
	return g.y
}

func (g grid) GetEditCellAddress() *Address {
	if g.editCell == nil {
		return nil
	}
	return &Address{g.editCell.Row, g.editCell.Col}
}

func (g *grid) AddContainer(container Container) {
	g.container = container
}

func (g grid) GetElement() js.Value {
	return g.main
}

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
func applyCss(el js.Value, class string) {
	list := css[class]
	style := ""
	for i := range list {
		style += list[i] + ";"
	}
	el.Set("style", style)
}

// Convert the screen coordinates to the grid row and col.
func (g grid) getLocation(x, y int) (int, int) {
	row := y / g.cellHeight
	col := x / g.cellWidth
	return row, col
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

// Draw an individual grid cell. If there is a container allow it to set
// the cell and font styles.
// TODO: Consider making the fgColor and fontColor fields of the grid
// struct so that there can be a single call to the container.SetCellStyles
// that sets both for the grid.
func (c *Cell) draw() {
	// Set default cell styles.
	c.Grid.ctx.Set("font", "15px arial")
	fgColor := "white"
	c.Grid.ctx.Set("fillStyle", fgColor)
	// Notify the container that the cell is being drawn so any custom 
	// cell styles can be applied to the canvas ctx.
	if c.Grid.container != nil {
		c.Grid.container.SetCellStyles(c.Row, c.Col)
	}
	fgColor = c.Grid.ctx.Get("fillStyle").String()
	// If the background is white no need to fill the rect.
	if fgColor != "#ffffff" {
		c.Grid.ctx.Call("fillRect", c.X-c.Grid.x, c.Y-c.Grid.y, c.Grid.cellWidth, c.Grid.cellHeight)
		// TODO: the default grid borders are lightgray consider making a setting and apply
		// the strokeStyle setting at the createBackGround call.
		c.Grid.ctx.Set("strokeStyle", "lightgray")
		c.Grid.ctx.Call("strokeRect", c.X-c.Grid.x, c.Y-c.Grid.y, c.Grid.cellWidth, c.Grid.cellHeight)
	}
	str := c.Value
	if c != c.Grid.editCell {
		for width := c.Grid.cellWidth + 1; width > c.Grid.cellWidth; {
			tm := c.Grid.ctx.Call("measureText", str)
			width = tm.Get("width").Int()
			if width > c.Grid.cellWidth {
				str = str[:len(str)-1]
			}
		}
	}
	fontColor := "black"
	c.Grid.ctx.Set("fillStyle", fontColor)
	// Notify the container that the cell is being drawn so any custom 
	// font styles can be applied to the canvas ctx.
	if c.Grid.container != nil {
		c.Grid.container.SetCellFontStyles(c.Row, c.Col)
	}
	c.Grid.ctx.Call("fillText", str, c.X-c.Grid.x, c.Y-c.Grid.y+15)
}

// Draw the grid foreground objects.
func (g grid) draw() {
	w := g.width
	h := g.height

	// Clip background canvas.
	g.ctx.Call("drawImage", g.cnv, g.sx, g.sy, w, h, 0, 0, w, h)

	g.ctx.Call("save")
	// Draw the data cells.
	for i := range g.data {
		// Edit cell may or may not be added to data cells yet.
		// Don't double draw.
		if g.data[i] != g.editCell {
			g.data[i].draw()
		}
	}

	// Draw the edit cell.
	if g.editCell != nil {
		g.editCell.draw()
	}
	g.ctx.Call("restore")

	// Draw the selected cells.
	g.ctx.Call("save")
	g.ctx.Set("lineWidth", 1)
	for i := range g.selectedCells {
		g.ctx.Call("save")
		s := g.selectedCells[i]
		shadowColor := "blue"
		borderColor := "lightblue"
		if s.Editing {
			shadowColor = "green"
			borderColor = "lightgreen"
		}
		g.ctx.Set("shadowColor", shadowColor)
		g.ctx.Set("strokeStyle", borderColor)
		g.ctx.Set("shadowBlur", 2)
		g.ctx.Call("strokeRect", s.X-g.x+2, s.Y-g.y+2, g.cellWidth-2, g.cellHeight-2)
		g.ctx.Call("restore")
	}
	g.ctx.Call("restore")

	// Draw the scroll controls.
	g.ctx.Call("save")
	g.ctx.Set("strokeStyle", "gray")
	g.ctx.Set("fillStyle", "lightgray")
	g.ctx.Call("fillRect", 0, h-g.cellHeight, g.cellWidth/2, g.cellHeight)
	g.ctx.Call("strokeRect", 0, h-g.cellHeight, g.cellWidth/2, g.cellHeight)
	g.ctx.Call("fillRect", w-g.cellWidth/2, h-g.cellHeight, g.cellWidth/2, g.cellHeight)
	g.ctx.Call("strokeRect", w-g.cellWidth/2, h-g.cellHeight, g.cellWidth/2, g.cellHeight)
	g.ctx.Call("fillRect", w-g.cellWidth/2, 0, g.cellWidth/2, g.cellHeight)
	g.ctx.Call("strokeRect", w-g.cellWidth/2, 0, g.cellWidth/2, g.cellHeight)
	g.ctx.Call("fillRect", w-g.cellWidth/2, h-g.cellHeight*2, g.cellWidth/2, g.cellHeight)
	g.ctx.Call("strokeRect", w-g.cellWidth/2, h-g.cellHeight*2, g.cellWidth/2, g.cellHeight)
	g.ctx.Call("restore")
}

// Helper for creating dom elements.
func CreateElement(typ string) js.Value {
	doc := js.Global().Get("document")
	return doc.Call("createElement", typ)
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

// Move the grid's viewport.
func (g *grid) move(dx, dy int) bool {
	// left and top edges.
	if dx < 0 && g.x+dx < 0 {
		g.x = 0
		g.sx = 0
		g.direction = -1
		g.scrolling = false
		js.Global().Call("clearInterval", g.interval)
		return false
	}
	if dy < 0 && g.y+dy < 0 {
		g.y = 0
		g.sy = 0
		g.direction = -1
		g.scrolling = false
		js.Global().Call("clearInterval", g.interval)
		return false
	}

	g.x += dx
	g.y += dy

	g.sx += dx
	g.sy += dy

	// Recycle background canvas offsets.
	if g.direction < 2 && g.x%g.width == 0 {
		if g.direction == 0 {
			g.sx = 0
		} else if g.direction == 1 {
			g.sx = g.width
		}
	}
	if g.direction > 1 && g.y%g.height == 0 {
		if g.direction == 2 {
			g.sy = 0
		} else if g.direction == 3 {
			g.sy = g.height
		}
	}

	return true
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

func getBounds(cnv js.Value) (int, int) {
	bounds := cnv.Call("getBoundingClientRect")
	x := bounds.Get("left").Int()
	y := bounds.Get("top").Int()
	return x, y
}

func getScrollCoords() (int, int) {
	x := js.Global().Get("window").Get("scrollX").Int()
	y := js.Global().Get("window").Get("scrollY").Int()
	return x, y
}

// Convert screen coordinates to an Address.
func (g *grid) getAddress(x, y int) (Address, int, int) {
	var a Address
	bx, by := getBounds(g.vcnv)
	x -= bx
	y -= by
	x += g.x
	y += g.y
	wx, wy := getScrollCoords()
	dx := x / g.cellWidth
	x -= wx
	y -= wy
	sx := g.cellWidth * dx
	dy := y / g.cellHeight
	sy := g.cellHeight * dy
	row, col := g.getLocation(sx, sy)
	a.Row = row
	a.Col = col
	return a, sx, sy
}

// Select a grid cell by screen coordinates.
func (g *grid) selectCell(x, y int) *Cell {
	a, sx, sy := g.getAddress(x, y)
	if s, ok := g.data[a]; ok {
		g.selectedCells[a] = s
		return s
	}
	if s, ok := g.selectedCells[a]; ok {
		return s
	}
	s := Cell{sx, sy, a.Row, a.Col, "", false, g}//, nil}
	g.selectedCells[a] = &s
	return &s
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

// Convert row and col values to screen coordinates.
func (g *grid) addressToCoords(row, col int) (int, int) {
	x := col * g.cellWidth
	y := row * g.cellHeight

	/*x += g.x
	y += g.y*/

	return x, y
}

// Add a value to the cell at the Address of row and col of the grid.
func (g *grid) addData(row, col int, value string) *Cell { //, cl *xlsx.Cell) {
	if c, ok := g.data[Address{row, col}]; ok {
		c.Value = value
		if g.container != nil {
			g.container.AddCell(c)
		}
		return c
	}
	x, y := g.addressToCoords(row, col)
	c := Cell{x, y, row, col, value, false, g}
	g.data[Address{row, col}] = &c
	if g.container != nil {
		fmt.Println("adding cell")
		g.container.AddCell(&c)
	}
	return &c
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

func NewGrid(obj GridObj) Grid {
	main := CreateElement("div")
	//js.Global().Get("document").Get("body").Call("appendChild", main)
	ctx, vcnv := createView(obj.width, obj.height, main)
	applyCss(vcnv, obj.class)
	cnv := createBackGround(obj.width, obj.height, obj.cellWidth, obj.cellHeight)
	g := grid{obj.class,
	0,
	0,
	0,
	0,
	obj.width,
	obj.height,
	vcnv,
	cnv,
	ctx,
	main,
	map[Address]*Cell{},
	map[Address]*Cell{},
	obj.cellWidth,
	obj.cellHeight,
	-1,
	js.Value{},
	obj.speed,
	nil,
	false,
	false,
	false,
	0,
	0,
	nil}

	grids[obj.id] = &g

	// Interval callback to handle scrolling while mouse button is down.
	moveCb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		switch g.direction {
		case 0:
			g.move(5, 0)
		case 1:
			g.move(-5, 0)
		case 2:
			g.move(0, 5)
		case 3:
			g.move(0, -5)
		}
		g.scrollAmt--
		g.Draw()
		return nil
	})

	// Interval callback to handle scrolling from scroll event.
	moveAmtCb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		switch g.direction {
		case 0:
			g.move(5, 0)
		case 1:
			g.move(-5, 0)
		case 2:
			g.move(0, 5)
		case 3:
			g.move(0, -5)
		}
		g.Draw()
		g.scrollAmt--
		if g.scrollAmt == 0 {
			js.Global().Call("clearInterval", g.interval)
			g.scrolling = false
		}
		return nil
	})


	// Handle clicks on grid's view canvas area.
	mouseDownCb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		e := args[0]
		x := e.Get("pageX").Int()
		y := e.Get("pageY").Int()
		bx, by := getBounds(vcnv)
		wx, wy := getScrollCoords()

		// Check if one of the scroll buttons was clicked.
		if x-bx-wx > g.width-g.cellWidth && x-bx-wx < g.width &&
		y-by-wy > g.height-g.cellHeight && y-by-wy < g.height {
			if g.move(5, 0) {
				g.direction = 0
				g.interval = js.Global().Call("setInterval", moveCb, g.speed)
			}
			return nil
		}
		if x-bx-wx > 0 && x-bx-wx < g.cellWidth &&
		y-by-wy > g.height-g.cellHeight && y-by-wy < g.height {
			if g.move(-5, 0) {
				g.direction = 1
				g.interval = js.Global().Call("setInterval", moveCb, g.speed)
			}
			return nil
		}

		if x-bx-wx > g.width-g.cellWidth && x-bx-wx < g.width &&
		y-by-wy > g.height-g.cellHeight*2 && y-by-wy < g.height-g.cellHeight {
			if g.move(0, 5) {
				g.direction = 2
				g.interval = js.Global().Call("setInterval", moveCb, g.speed)
			}
			return nil
		}
		if x-bx-wx > g.width-g.cellWidth && x-bx-wx < g.width &&
		y-by-wy > 0 && y-by-wy < g.cellHeight {
			if g.move(0, -5) {
				g.direction = 3
				g.interval = js.Global().Call("setInterval", moveCb, g.speed)
			}
			return nil
		}

		// Remove all selections.
		g.selectedCells = map[Address]*Cell{}
		g.editCell = nil
		g.selectCell(x, y)
		g.mouseDown = true
		g.Draw()
		return nil
	})

	// Clear the scroll interval. Used when mouse button goes
	// from down to up or when mouse leaves canvas area.
	mouseUpCb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		js.Global().Call("clearInterval", g.interval)
		g.direction = -1
		g.mouseDown = false
		return nil
	})

	// Activate a grid cell for editing.
	dblClickCb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		e := args[0]
		x := e.Get("pageX").Int()
		y := e.Get("pageY").Int()
		c := g.selectCell(x, y)
		c.Editing = true
		if g.editCell != nil {
			g.editCell.Editing = false
		}
		g.editCell = c
		g.Draw()
		return nil
	})

	// Handle keyboard input. Used for cell editing.
	keyDownCb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		e := args[0]
		c := e.Get("key").String()
		if g.active && !g.scrolling {
			switch c {
			case "ArrowRight":
				if g.move(5, 0) {
					g.scrolling = true
					g.direction = 0
					g.interval = js.Global().Call("setInterval", moveCb, g.speed)
				}
				return nil
			case "ArrowLeft":
				if g.move(-5, 0) {
					g.scrolling = true
					g.direction = 1
					g.interval = js.Global().Call("setInterval", moveCb, g.speed)
				}
				return nil
			case "ArrowDown":
				if g.move(0, 5) {
					g.scrolling = true
					g.direction = 2
					g.interval = js.Global().Call("setInterval", moveCb, g.speed)
				}
				return nil
			case "ArrowUp":
				if g.move(0, -5) {
					g.scrolling = true
					g.direction = 3
					g.interval = js.Global().Call("setInterval", moveCb, g.speed)
				}
				return nil
			}
		}
		if g.editCell != nil {
			e.Call("preventDefault")
			ec := g.editCell
			/*if _, ok := g.data[Address{ec.Row, ec.Col}]; !ok {
				g.AddData(ec.Row, ec.Col, ec.Value)
			}*/
			if c == "Tab" {
				delete(g.selectedCells, Address{ec.Row, ec.Col})
				g.AddData(ec.Row, ec.Col, ec.Value)
				ec.Editing = false
				g.editCell = nil
			} else if c == "Backspace" {
				if len(g.editCell.Value) > 0 {
					g.editCell.Value = g.editCell.Value[:len(g.editCell.Value)-1]
				}
			} else if len(c) == 1 {
				if g.editCell.Editing {
					g.editCell.Value += c
				} else {
					g.editCell.Value = c
					g.editCell.Editing = true
				}
			}
		}

		g.Draw()
		return nil
	})

	keyUpCb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if g.scrolling {
			js.Global().Call("clearInterval", g.interval)
			g.scrolling = false
		}
		return nil
	})

	mouseEnterCb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		g.active = true
		return nil
	})

	mouseLeaveCb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		js.Global().Call("clearInterval", g.interval)
		g.direction = -1
		g.scrolling = false
		g.active = false
		return nil
	})

	mouseMoveCb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if g.active && g.mouseDown {
			e := args[0]
			x := e.Get("pageX").Int()
			y := e.Get("pageY").Int()
			a, _, _ := g.getAddress(x, y)
			if _, ok := g.selectedCells[a]; !ok {
				g.selectCell(x, y)
				g.Draw()
			}
		}
		return nil
	})

	scrollCb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		e := args[0]
		e.Call("preventDefault")
		if g.active && !g.scrolling {
			g.scrolling = true
			js.Global().Get("window").Call("requestAnimationFrame", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				d := js.Global().Get("document")
				body := d.Get("body")
				el := d.Get("documentElement")
				g.scrollAmt = 50
				scroll := 5
				if body.Get("scrollTop").Int() > g.lastScroll || el.Get("scrollTop").Int() > g.lastScroll {
					fmt.Println("scroll down")
					g.direction = 2
				} else {
					fmt.Println("scroll up")
					g.direction = 3
					scroll = -5
				}
				if g.move(0, scroll) {
					g.interval = js.Global().Call("setInterval", moveAmtCb, g.speed)
				}
				return nil
			}))
		}
		return nil
	})

	vcnv.Call("addEventListener", "mousedown", mouseDownCb)
	vcnv.Call("addEventListener", "mouseup", mouseUpCb)
	vcnv.Call("addEventListener", "dblclick", dblClickCb)
	vcnv.Call("addEventListener", "mouseenter", mouseEnterCb)
	vcnv.Call("addEventListener", "mouseleave", mouseLeaveCb)
	vcnv.Call("addEventListener", "mousemove", mouseMoveCb)
	js.Global().Get("document").Call("addEventListener", "scroll", scrollCb)
	js.Global().Get("document").Call("addEventListener", "keydown", keyDownCb)
	js.Global().Get("document").Call("addEventListener", "keyup", keyUpCb)

	return &g
}

// External JavaScript function to create a new grid.
// args: JSON object(GridObj).
func NewGridJs(this js.Value, args[]js.Value) interface{} {
	obj := NewGridObj(args[0])
	g := NewGrid(obj)
	g.Draw()
	return g.GetElement()
}
