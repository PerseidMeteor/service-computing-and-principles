//client for restful server

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type Dev struct {
	ID          string `json:"id"`
	DevName     string `json:"devname"`
	Description string `json:"description"`
	Location    string `json:"location"`
	Status      string `json:"status"`
}

func main() {
	//get all devs
	getAllDevs()

	//get dev by id
	getDevByID("1")

	//post a new dev
	postDev()

	//get dev by id
	getDevByID("4")

	//put a new dev
	putDev()

	//get dev by id
	getDevByID("4")

	//delete dev by id
	deleteDevByID("4")

	//get dev by id
	getDevByID("4")
}

func getAllDevs() {
	// Send a GET request to http://localhost:8080/devs
	resp, err := http.Get("http://localhost:8080/devs")
	if err != nil {
		fmt.Println("Error reading response. ", err)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading body. ", err)
	}

	fmt.Println(string(body))
}

func getDevByID(id string) {
	// Send a GET request to http://localhost:8080/devs/1
	resp, err := http.Get("http://localhost:8080/devs/" + id)
	if err != nil {
		fmt.Println("Error reading response. ", err)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading body. ", err)
	}

	fmt.Println(string(body))
}

func postDev() {
	// Create a new dev struct.
	dev := Dev{ID: "4", DevName: "keyboard", Description: "Acer keyboard", Location: "QiXiang 318", Status: "fine"}

	// Convert the struct to JSON.
	jsonReq, err := json.Marshal(dev)
	if err != nil {
		fmt.Println("Error marshalling data. ", err)
	}

	// Send a POST request with JSON as the body.
	resp, err := http.Post("http://localhost:8080/devs", "application/json", bytes.NewBuffer(jsonReq))
	if err != nil {
		fmt.Println("Error reading response. ", err)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading body. ", err)
	}

	fmt.Println(string(body))
}

func putDev() {
	// Create a new dev struct.
	dev := Dev{ID: "4", DevName: "keyboard", Description: "Acer keyboard", Location: "QiXiang 318", Status: "bad"}

	// Convert the struct to JSON.
	jsonReq, err := json.Marshal(dev)
	if err != nil {
		fmt.Println("Error marshalling data. ", err)
	}

	// Send a PUT request with JSON as the body.
	req, err := http.NewRequest(http.MethodPut, "http://localhost:8080/devs/4", bytes.NewBuffer(jsonReq))
	if err != nil {
		fmt.Println("Error creating request. ", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Error reading response. ", err)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading body. ", err)
	}

	fmt.Println(string(body))
}

func deleteDevByID(id string) {
	// Send a DELETE request to http://localhost:8080/devs/1
	req, err := http.NewRequest(http.MethodDelete, "http://localhost:8080/devs/"+id, nil)
	if err != nil {
		fmt.Println("Error creating request. ", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Error reading response. ", err)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading body. ", err)
	}

	fmt.Println(string(body))
}
