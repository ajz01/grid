package main

import (
	"github.com/ajz01/grid"
	"syscall/js"
)

func main() {
	c := make(chan bool)
	js.Global().Set("newGrid", js.FuncOf(grid.NewGridJs))
	js.Global().Set("setCssMap", js.FuncOf(grid.SetCssMap))
	js.Global().Set("addData", js.FuncOf(grid.AddData))
	<-c
}
