module test

go 1.27.3

require hexago v0.0.0

require (
	golang.org/x/image v0.48.0 // indirect
	golang.org/x/sys v0.50.0 // indirect
)

// Движок пока не опубликован, поэтому берётся из папки на диске.
replace hexago => C:/dev/projects/gexago/engine
