// Package grid provides the implementation of a wasm based grid control.
// The control can be created and modified from JavaScript code by using
// the public API methods defined in the js.go file. The grid package will
// handle the basic requirements for the grid such as drawing and event
// handling. To add additional drawing or event handling features a Container
// interface is provided. The Container interface provides "hooks" into the
// drawing methods of the grid to allow any implementing type an oppurtunity
// to add additional layers to the basic drawing method. An implementation of
// Container can also add additional JavaScript event handlers to the grid.
package grid

import (
	"syscall/js"
)

// grid scroll directions.
type direction int

const (
	none direction = -1
	right direction = iota
	left
	down
	up
)

// The main grid type.
type grid struct {
	class          string
	x, y, sx, sy   int
	width, height  int
	vcnv, cnv, ctx js.Value
	main           js.Value
	selectedCells  map[Address]*cell
	data           map[Address]*cell
	cellWidth      int
	cellHeight     int
	direction      direction // scroll direction
	interval       js.Value // scroll timer interval
	speed          int // scroll speed.
	editCell       *cell
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
	GetElement() *js.Value
	AddData(row, col int, value string)
	GetContainer() Container
	SelectCells([]Address)
	ClearSelection()
}

// The Container interface provides the methods for the grid.container.
// The grid.container allows for adding event handlers and cell styles
// to the grid while letting the grid handle the standard events and
// cell styles.
type Container interface {
	AddCell(cell CellContent)
	AddCellsDone()
	SetCellStyles(row, col int)
	SetCellFontStyles(row, col int)
	GetGrid() Grid
}

func (g *grid) Draw() {
	g.draw()
}

func (g *grid) AddData(row, col int, value string) {
	g.addData(row, col, value)
}

func (g *grid) SelectCells(addresses []Address) {
	for _, a := range addresses {
		x, y := g.addressToCoords(a.Row, a.Col)
		g.selectCellAddress(a, x, y)
	}
	g.draw()
}

func (g *grid) ClearSelection() {
	g.selectedCells = map[Address]*cell{}
}

func (g *grid) AddEventHandler(event string, handler func(this js.Value, args []js.Value) interface{}) {
	g.vcnv.Call("addEventListener", event, js.FuncOf(handler))
}

func (g *grid) GetCtx() *js.Value {
	return &g.ctx
}

func (g *grid) GetCellWidth() int {
	return g.cellWidth
}

func (g *grid) GetCellHeight() int {
	return g.cellHeight
}

func (g *grid) GetX() int {
	return g.x
}

func (g *grid) GetY() int {
	return g.y
}

func (g *grid) GetEditCellAddress() *Address {
	if g.editCell == nil {
		return nil
	}
	return &Address{g.editCell.row, g.editCell.col}
}

func (g *grid) AddContainer(container Container) {
	g.container = container
}

func (g *grid) GetContainer() Container {
	return g.container
}

func (g *grid) GetElement() *js.Value {
	return &g.main
}

// Convert the screen coordinates to the grid row and col.
func (g *grid) getLocation(x, y int) (int, int) {
	row := y / g.cellHeight
	col := x / g.cellWidth
	return row, col
}

// Draw the grid foreground objects.
func (g *grid) draw() {
	w := g.width
	h := g.height

	// Clip background canvas.
	g.ctx.Call("drawImage", g.cnv, g.sx, g.sy, w, h, 0, 0, w, h)

	// Draw the data cells.
	g.ctx.Call("save")
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
		s := g.selectedCells[i]
		shadowColor := "blue"
		borderColor := "lightblue"
		if s.editing {
			shadowColor = "green"
			borderColor = "lightgreen"
		}
		g.ctx.Set("shadowColor", shadowColor)
		g.ctx.Set("strokeStyle", borderColor)
		g.ctx.Set("shadowBlur", 2)
		g.ctx.Call("strokeRect", s.x-g.x+2, s.y-g.y+2, g.cellWidth-2, g.cellHeight-2)
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

// Move the grid's viewport.
func (g *grid) move(dx, dy int) bool {
	// Attempting to move outside left and top boundaries.
	if dx < 0 && g.x+dx < 0 {
		g.x = 0
		g.sx = 0
		g.direction = none
		g.scrolling = false
		js.Global().Call("clearInterval", g.interval)
		return false
	}
	if dy < 0 && g.y+dy < 0 {
		g.y = 0
		g.sy = 0
		g.direction = none
		g.scrolling = false
		js.Global().Call("clearInterval", g.interval)
		return false
	}

	g.x += dx
	g.y += dy

	g.sx += dx
	g.sy += dy

	// Recycle background canvas offsets.
	if g.direction < down && g.x%g.width == 0 {
		if g.direction == right {
			g.sx = 0
		} else if g.direction == left {
			g.sx = g.width
		}
	}
	if g.direction > left && g.y%g.height == 0 {
		if g.direction == down {
			g.sy = 0
		} else if g.direction == up {
			g.sy = g.height
		}
	}

	return true
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

func (g *grid) selectCellAddress(a Address, sx, sy int) *cell {
	if s, ok := g.data[a]; ok {
		g.selectedCells[a] = s
		return s
	}
	if s, ok := g.selectedCells[a]; ok {
		return s
	}
	s := cell{sx, sy, a.Row, a.Col, "", false, g}
	g.selectedCells[a] = &s
	return &s
}

// Select a grid cell by screen coordinates.
func (g *grid) selectCell(x, y int) *cell {
	a, sx, sy := g.getAddress(x, y)
	return g.selectCellAddress(a, sx, sy)
}

// Convert row and col values to screen coordinates.
func (g *grid) addressToCoords(row, col int) (int, int) {
	x := col * g.cellWidth
	y := row * g.cellHeight

	return x, y
}

// Add a value to the cell at the Address of row and col of the grid.
func (g *grid) addData(row, col int, value string) *cell {
	if c, ok := g.data[Address{row, col}]; ok {
		c.value = value
		if g.container != nil {
			g.container.AddCell(c)
		}
		return c
	}
	x, y := g.addressToCoords(row, col)
	c := cell{x, y, row, col, value, false, g}
	g.data[Address{row, col}] = &c
	if g.container != nil {
		g.container.AddCell(&c)
	}
	return &c
}

func NewGrid(obj GridObj) Grid {
	// Create a div to add the grid to.
	main := CreateElement("div")
	ctx, vcnv := createView(obj.width, obj.height, main)
	ApplyCss(&vcnv, obj.class)

	cnv := createBackGround(obj.width, obj.height, obj.cellWidth, obj.cellHeight)

	g := grid{obj.class, 0, 0, 0, 0, obj.width, obj.height, vcnv, cnv, ctx,
	main, map[Address]*cell{}, map[Address]*cell{}, obj.cellWidth, obj.cellHeight,
	-1, js.Value{}, obj.speed, nil, false, false, false, 0, 0, nil}

	grids[obj.id] = &g

	// Interval callback to handle continued scrolling while mouse button is down.
	moveCb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		switch g.direction {
		case right:
			g.move(5, 0)
		case left:
			g.move(-5, 0)
		case down:
			g.move(0, 5)
		case up:
			g.move(0, -5)
		}
		g.scrollAmt--
		g.Draw()
		return nil
	})

	// Interval callback to handle continued scrolling from scroll event.
	moveAmtCb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		switch g.direction {
		case right:
			g.move(5, 0)
		case left:
			g.move(-5, 0)
		case down:
			g.move(0, 5)
		case up:
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
				g.direction = right
				g.interval = js.Global().Call("setInterval", moveCb, g.speed)
			}
			return nil
		}
		if x-bx-wx > 0 && x-bx-wx < g.cellWidth &&
		y-by-wy > g.height-g.cellHeight && y-by-wy < g.height {
			if g.move(-5, 0) {
				g.direction = left
				g.interval = js.Global().Call("setInterval", moveCb, g.speed)
			}
			return nil
		}

		if x-bx-wx > g.width-g.cellWidth && x-bx-wx < g.width &&
		y-by-wy > g.height-g.cellHeight*2 && y-by-wy < g.height-g.cellHeight {
			if g.move(0, 5) {
				g.direction = down
				g.interval = js.Global().Call("setInterval", moveCb, g.speed)
			}
			return nil
		}
		if x-bx-wx > g.width-g.cellWidth && x-bx-wx < g.width &&
		y-by-wy > 0 && y-by-wy < g.cellHeight {
			if g.move(0, -5) {
				g.direction = up
				g.interval = js.Global().Call("setInterval", moveCb, g.speed)
			}
			return nil
		}

		// Remove all selections.
		g.selectedCells = map[Address]*cell{}
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
		g.direction = none
		g.mouseDown = false
		return nil
	})

	// Activate a grid cell for editing.
	dblClickCb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		e := args[0]
		x := e.Get("pageX").Int()
		y := e.Get("pageY").Int()
		c := g.selectCell(x, y)
		c.editing = true
		if g.editCell != nil {
			g.editCell.editing = false
		}
		g.editCell = c
		g.Draw()
		return nil
	})

	// Handle keyboard input. Used for cell editing.
	editing := false
	keyDownCb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		e := args[0]
		c := e.Get("key").String()
		if g.active && !g.scrolling {
			switch c {
			case "ArrowRight":
				if g.move(5, 0) {
					g.scrolling = true
					g.direction = right
					g.interval = js.Global().Call("setInterval", moveCb, g.speed)
				}
				return nil
			case "ArrowLeft":
				if g.move(-5, 0) {
					g.scrolling = true
					g.direction = left
					g.interval = js.Global().Call("setInterval", moveCb, g.speed)
				}
				return nil
			case "ArrowDown":
				if g.move(0, 5) {
					g.scrolling = true
					g.direction = down
					g.interval = js.Global().Call("setInterval", moveCb, g.speed)
				}
				return nil
			case "ArrowUp":
				if g.move(0, -5) {
					g.scrolling = true
					g.direction = up
					g.interval = js.Global().Call("setInterval", moveCb, g.speed)
				}
				return nil
			}
		}
		if g.editCell != nil {
			e.Call("preventDefault")
			ec := g.editCell
			if c == "Tab" {
				delete(g.selectedCells, Address{ec.row, ec.col})
				g.AddData(ec.row, ec.col, ec.value)
				if g.container != nil {
					g.container.AddCellsDone()
				}
				ec.editing = false
				g.editCell = nil
				editing = false
			} else if c == "Backspace" {
				if len(g.editCell.value) > 0 {
					g.editCell.value = g.editCell.value[:len(g.editCell.value)-1]
				}
			} else if len(c) == 1 {
				if g.editCell.editing {
					// If not editing then this is the first
					// letter. Clear the existing cell contents.
					if !editing {
						g.editCell.value = ""
					}
					editing = true
					g.editCell.value += c
				} else {
					g.editCell.value = c
					g.editCell.editing = true
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
		g.direction = none
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
					g.direction = down
				} else {
					g.direction = up
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
