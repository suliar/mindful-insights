package main

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	fmt.Println("Hello world")

	r := gin.Default()

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"mindful": "insights"})
	})

	if err := r.Run(); err != nil {
		log.Fatal(err)
	}
}
