package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
)

type Envelope struct {
	XMLName       xml.Name `xml:"soap:Envelope"`
	Soap          string   `xml:"xmlns:soap,attr"`
	EncodingStyle string   `xml:"soap:encodingStyle,attr"`
	Body          Body     `xml:"soap:Body"`
}

type Body struct {
	XMLName  xml.Name            `xml:"soap:Body"`
	N        string              `xml:"xmlns:n,attr"`
	Response *GetWeatherResponse `xml:"m:GetWeatherResponse,omitempty"`
	Request  *GetWeather         `xml:"n:GetWeather,omitempty"`
}

type GetWeather struct {
	CityName string `xml:"CityName,omitempty"`
}

type GetWeatherResponse struct {
	Temperature float32 `xml:"m:Temperature,omitempty"`
	Weather     string  `xml:"m:Weather,omitempty"`
}

func main() {

	request := GetWeather{"西安"}

	body := Body{XMLName: xml.Name{Space: "", Local: "soap:Body"}, N: "http://www.nwpu.edu.cn/soa/xml/test", Request: &request}

	env := Envelope{xml.Name{Space: "", Local: "Envelope"},
		"http://www.w3.org/2001/12/soap-envelope",
		"http://www.w3.org/2001/12/soap-encoding",
		body}

	data, err := xml.MarshalIndent(env, "  ", "    ")

	fmt.Println("data:", string(data))

	//post xml to server
	resp, err := http.Post("http://localhost:8081/weather",
		"application/xml",
		ioutil.NopCloser(bytes.NewBuffer(data)))

	if err != nil {
		fmt.Println(err)
	}
	defer resp.Body.Close()

	//读取body内的内容
	content, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(content))

}
