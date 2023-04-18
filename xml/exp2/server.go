package main

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

// TODO:make full GetWeather struct
type GetWeather struct {
	CityName string `xml:"CityName"`
}

func main() {
	//注册回调函数
	http.HandleFunc("/weather", handleGetWeather)
	log.Println("weather server start...")
	//绑定tcp监听地址，并开始接受请求，然后调用服务端处理程序来处理传入的连接请求
	//参数1为addr即监听地址；参数2表示服务端处理程序，通常为nil
	//当参数2为nil时，服务端调用http.DefaultServeMux进行处理
	err := http.ListenAndServe(":8081", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

// TODO:get true weather
func getCityWeather(CityName string) (string, error) {

	return "Sunny", nil
}
func handleGetWeather(w http.ResponseWriter, r *http.Request) {
	// 读取请求体中的XML数据
	xmlData, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Println("Error reading XML data:", err)
		return
	}

	// 解析XML数据到Person结构体中
	var gw GetWeather

	err = xml.Unmarshal(xmlData, &gw)
	if err != nil {
		fmt.Println("Error unmarshalling XML data:", err)
		return
	}

	// 输出解析后的Person对象
	fmt.Println("Received city:", gw.CityName)

	// 获取天气
	weather, err := getCityWeather(gw.CityName)
	if err != nil {
		fmt.Println("Error get city weather:", err)
		return
	}

	// 返回响应
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(weather))
}
