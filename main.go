package main

import (
	"github.com/jjhwan-h/bundle-server/cmd"
	_ "github.com/jjhwan-h/bundle-server/docs"
)

// @title       bundle server
// @version     1.0
// @description bundle server
// @BasePath   	/

func main() {
	cmd.Execute()
}
