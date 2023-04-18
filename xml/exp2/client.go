package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
)

// TODO:make full GetWeather struct
type GetWeather struct {
	CityName string `xml:"CityName"`
}

func main() {

	werther := GetWeather{CityName: "xian"}
	data, err := xml.Marshal(werther)

	fmt.Println("data:", string(data))

	//post xml to server
	resp, err := http.Post("http://localhost:8081/weather",
		"application/xml",
		ioutil.NopCloser(bytes.NewBuffer(data)))

	if err != nil {
		fmt.Println(err)
	}
	defer resp.Body.Close()
	// 获取服务器端读到的数据
	fmt.Println("Status = ", resp.Status)         // 状态
	fmt.Println("StatusCode = ", resp.StatusCode) // 状态码
	fmt.Println("Header = ", resp.Header)         // 响应头部
	fmt.Println("Body = ", resp.Body)             // 响应包体
	//读取body内的内容
	content, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(content))

}
