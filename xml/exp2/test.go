package main

import (
	"fmt"

	"github.com/beevik/etree"
)

func main() {
	xmlStr := ` <soap:Envelope xmlns:soap="http://www.w3.org/2001/12/soap-envelope" soap:encodingStyle="http://www.w3.org/2001/12/soap-encoding">
	<soap:Body xmlns:n="http://www.nwpu.edu.cn/soa/xml/test">
		<n:GetWeather>
			<CityName>西安</CityName>
		</n:GetWeather>
	</soap:Body>
</soap:Envelope>`

	doc := etree.NewDocument()
	if err := doc.ReadFromString(xmlStr); err != nil {
		panic(err)
	}

	cityName := doc.FindElement("//CityName")
	if cityName == nil {
		panic("CityName element not found")
	}

	value := cityName.Text()
	fmt.Println(value) // 输出：西安
}
