package main

import (
	"net/http"

	gin "github.com/gin-gonic/gin"
)

type Dev struct {
	ID          string `json:"id"`
	DevName     string `json:"devname"`
	Description string `json:"description"`
	Location    string `json:"location"`
	Status      string `json:"status"`
}

var devs = []Dev{
	Dev{ID: "1", DevName: "computer", Description: "LianXiang workstation", Location: "QiXiang 318", Status: "fine"},
	Dev{ID: "2", DevName: "laptop", Description: "Acer laptop", Location: "QiXiang 318", Status: "fine"},
	Dev{ID: "3", DevName: "mouse", Description: "Acer mouse", Location: "QiXiang 318", Status: "bad"},
}

// GetDevs responds with the list of all devs as JSON.
func GetAllDevs(c *gin.Context) {
	c.IndentedJSON(http.StatusOK, devs)
}

// PostDev adds an dev from JSON received in the request body.
func PostDev(c *gin.Context) {
	var newDev Dev

	// Call BindJSON to bind the received JSON to
	// newDev.
	if err := c.BindJSON(&newDev); err != nil {
		return
	}

	// Add the new dev to the slice.
	devs = append(devs, newDev)
	c.IndentedJSON(http.StatusCreated, newDev)
}

// GetDevByID locates the dev whose ID value matches the id
// parameter sent by the client, then returns that dev as a response.
func GetDevByID(c *gin.Context) {
	// Get the ID from the url.
	id := c.Param("id")

	// Loop over the list of devs, looking for
	// an dev whose ID value matches the parameter.
	for _, dev := range devs {
		if dev.ID == id {
			c.IndentedJSON(http.StatusOK, dev)
			return
		}
	}
	c.IndentedJSON(http.StatusNotFound, gin.H{"message": "dev not found"})
}

// PutDev updates an existing dev in the devs slice.
func PutDev(c *gin.Context) {
	// Get the ID from the url.
	id := c.Param("id")
	var newDev Dev

	// Call BindJSON to bind the received JSON to
	// newDev.
	if err := c.BindJSON(&newDev); err != nil {
		return
	}

	// Loop over the list of devs, looking for
	// an dev whose ID value matches the parameter.
	for index, dev := range devs {
		if dev.ID == id {
			devs[index] = newDev
			c.IndentedJSON(http.StatusOK, newDev)
			return
		}
	}
	c.IndentedJSON(http.StatusNotFound, gin.H{"message": "dev not found"})
}

// DeleteDev deletes an dev by ID.
func DeleteDev(c *gin.Context) {
	// Get the ID from the url.
	id := c.Param("id")

	// Loop through the list of devs.
	for index, dev := range devs {
		if dev.ID == id {
			devs = append(devs[:index], devs[index+1:]...)
			c.IndentedJSON(http.StatusOK, dev)
			return
		}
	}
	c.IndentedJSON(http.StatusNotFound, gin.H{"message": "dev not found"})
}

func main() {

	// Set the router as the default one shipped with Gin
	router := gin.Default()

	// Add, Get, Put, Delete Devs
	router.GET("/devs", GetAllDevs)
	router.GET("/devs/:id", GetDevByID)
	router.POST("/devs", PostDev)
	router.PUT("/devs/:id", PutDev)
	router.DELETE("/devs/:id", DeleteDev)

	router.Run("localhost:8080")
}
