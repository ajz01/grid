package grid

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

type CellContent interface {
	GetRow() int
	GetCol() int
	GetValue() string
	SetValue(v string)
}

func (c Cell) GetRow() int {
	return c.Row
}

func (c Cell) GetCol() int {
	return c.Col
}

func (c Cell) GetValue() string {
	return c.Value
}

func (c *Cell) SetValue(v string) {
	c.Value = v
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
