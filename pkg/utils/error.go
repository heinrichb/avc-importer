// pkg/utils/error.go
package utils

import (
	"fmt"
	"os"
)

/*
HandleError prints an error message and optionally exits the application.

Parameters:
  - err: The error object to handle.
  - exit: A boolean indicating if the program should terminate.

Usage:
    utils.HandleError(err, true)  // Exits program
    utils.HandleError(err, false) // Prints error, continues execution
*/
func HandleError(err error, exit bool) {
	if err != nil {
		fmt.Println("[Error]:", err.Error())
		if exit {
			os.Exit(1)
		}
	}
}
