package main

import (
	"log"

	"droneops-sim/internal/dashboard"
)

func main() {
	if err := dashboard.Render("build"); err != nil {
		log.Fatal(err)
	}
}
