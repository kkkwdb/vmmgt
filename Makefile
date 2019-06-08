version:=$(shell git describe --tags --dirty)
vmmgt:
	sed 's/app.Version =.*/app.Version = $(version)/' main.go
