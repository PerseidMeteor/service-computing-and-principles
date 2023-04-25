package main

import (
	"fmt"

	"github.com/beevik/etree"
)

func main() {
	// 创建根元素 xsd:schema
	xsd := etree.NewDocument()
	xsd.CreateProcInst("xml", `version="1.0" encoding="UTF-8"`)
	xsd.SetRoot(etree.NewElement("xsd:schema"))
	xsd.Root().CreateAttr("xmlns:xsd", "http://www.w3.org/2001/XMLSchema")
	xsd.Root().CreateAttr("elementForm Default", "qualified")

	// 创建 xsd:element 元素
	bookstore := xsd.CreateElement("xsd:element")
	bookstore.CreateAttr("name", "bookstore")
	bookstore.CreateAttr("type", "bookstoreType")

	// 创建 xsd:complexType 元素 bookstoreType
	bookstoreType := xsd.CreateElement("xsd:complexType")
	bookstoreType.CreateAttr("name", "bookstoreType")

	// 创建 bookstoreType 的 xsd:sequence 子元素
	sequence := bookstoreType.CreateElement("xsd:sequence")
	sequence.CreateAttr("maxOccurs", "unbounded")
	book := sequence.CreateElement("xsd:element")
	book.CreateAttr("name", "book")
	book.CreateAttr("type", "bookType")

	// 创建 xsd:complexType 元素 bookType
	bookType := xsd.CreateElement("xsd:complexType")
	bookType.CreateAttr("name", "bookType")

	// 创建 bookType 的 xsd:sequence 子元素
	bookSequence := bookType.CreateElement("xsd:sequence")
	title := bookSequence.CreateElement("xsd:element")
	title.CreateAttr("name", "title")
	title.CreateAttr("type", "xsd:string")
	author := bookSequence.CreateElement("xsd:element")
	author.CreateAttr("name", "author")
	author.CreateAttr("type", "authorName")
	price := bookSequence.CreateElement("xsd:element")
	price.CreateAttr("name", "price")
	price.CreateAttr("type", "xsd:decimal")

	// 创建 bookType 的 xsd:attribute 子元素
	genre := bookType.CreateElement("xsd:attribute")
	genre.CreateAttr("name", "genre")
	genre.CreateAttr("type", "xsd:string")

	// 创建 xsd:complexType 元素 authorName
	authorName := xsd.CreateElement("xsd:complexType")
	authorName.CreateAttr("name", "authorName")
	authorSequence := authorName.CreateElement("xsd:sequence")
	firstName := authorSequence.CreateElement("xsd:element")
	firstName.CreateAttr("name", "first-name")
	firstName.CreateAttr("type", "xsd:string")
	lastName := authorSequence.CreateElement("xsd:element")
	lastName.CreateAttr("name", "last-name")
	lastName.CreateAttr("type", "xsd:string")

	// 打印生成的 XML 文档
	xsd.Indent(2)
	content, _ := xsd.WriteToString()
	fmt.Print(content)
	fmt.Println("</xsd:schema> ")

}
