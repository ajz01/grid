package grid

// A cell of the grid.
type cell struct {
	x           int
	y           int
	row         int
	col         int
	value       string
	editing     bool
	grid	*grid
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

func (c *cell) GetRow() int {
	return c.row
}

func (c *cell) GetCol() int {
	return c.col
}

func (c *cell) GetValue() string {
	return c.value
}

func (c *cell) SetValue(v string) {
	c.value = v
}

// Draw an individual grid cell. If there is a container allow it to set
// the cell and font styles.
// TODO: Consider making the fgColor and fontColor fields of the grid
// struct so that there can be a single call to the container.SetCellStyles
// that sets both for the grid.
func (c *cell) draw() {
	// Set default cell styles.
	c.grid.ctx.Set("font", "15px arial")
	fgColor := "white"
	c.grid.ctx.Set("fillStyle", fgColor)

	// Notify the container that the cell is being drawn so any custom 
	// cell styles can be applied to the canvas ctx.
	if c.grid.container != nil {
		c.grid.container.SetCellStyles(c.row, c.col)
	}
	fgColor = c.grid.ctx.Get("fillStyle").String()

	// If the background is white no need to fill the rect.
	if fgColor != "#ffffff" {
		c.grid.ctx.Call("fillRect", c.x-c.grid.x, c.y-c.grid.y, c.grid.cellWidth, c.grid.cellHeight)

		// TODO: the default grid borders are lightgray consider making a setting and apply
		// the strokeStyle setting at the createBackGround call.
		c.grid.ctx.Set("strokeStyle", "lightgray")
		c.grid.ctx.Call("strokeRect", c.x-c.grid.x, c.y-c.grid.y, c.grid.cellWidth, c.grid.cellHeight)
	}
	str := c.value
	if c != c.grid.editCell {
		for width := c.grid.cellWidth + 1; width > c.grid.cellWidth; {
			tm := c.grid.ctx.Call("measureText", str)
			width = tm.Get("width").Int()
			if width > c.grid.cellWidth {
				str = str[:len(str)-1]
			}
		}
	}
	fontColor := "black"
	c.grid.ctx.Set("fillStyle", fontColor)

	// Notify the container that the cell is being drawn so any custom 
	// font styles can be applied to the canvas ctx.
	if c.grid.container != nil {
		c.grid.container.SetCellFontStyles(c.row, c.col)
	}
	c.grid.ctx.Call("fillText", str, c.x-c.grid.x, c.y-c.grid.y+15)
}
