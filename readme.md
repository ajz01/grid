# grid

A wasm based grid control built with go.

The repository contains a test server that will serve the gzipped wasm file.

The sample html code expects a wasm file named test.wasm.gz to be built in the wasm directory. This can be done using the following commands from the wasm directory:

GOOS=js GOARCH=wasm go build -o test.wasm

gzip -f test.wasm

If you want to use the test server to serve the compressed wasm file change to the server directory run go build to build the server (if it's not already built) and execute ./server which will listen on the localhost port 8080 by default.

Navigate to the modified go provided wasm_exec.html to view the grid from a browser.

localhost:8080/wasm_exec.html

The grid currently supports scrolling and has some basic scroll controls added to the display corners. Cells can be selected by clicking on the grid and dragging the mouse. Data can be added to the cells from JavaScript using the js api or by double clicking a cell and typing with the keyboard. The rows and columns are not bounded and neither is number of populated cells. The grid has a container field that can be used to extend the grid by adding additional event handlers or used to style the cell or font styles.

The features are still very limited as this is a new project, but it seems there is a lot of potential for building fully encapsulated 'web component' style controls using wasm and go makes it easy to build.

![Sample Image](/images/grid.png)
