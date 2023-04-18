package main

import (
	"encoding/xml"
	"os"
)

type Bookstore struct {
	XMLName xml.Name `xml:"bookstore"`
	Books   []Book   `xml:"book"`
	// Xsd      string   `xml:"xmlns:xsd,attr"`
	// Xsi      string   `xml:"xmlns,attr"`
	// XsiType  string   `xml:"xsi:type,attr"`
	// XsdValue string   `xml:"xsd:value,attr"`
}

type Book struct {
	XMLName xml.Name `xml:"book"`
	Genre   string   `xml:"genre,attr"`
	Title   string   `xml:"title"`
	Author  Author   `xml:"author"`
	Price   float64  `xml:"price"`
}

type Author struct {
	XMLName   xml.Name `xml:"author"`
	FirstName string   `xml:"first-name"`
	LastName  string   `xml:"last-name"`
}

func main() {
	bookstore := Bookstore{
		// Xsd:      "http://www.w3.org/2001/XMLSchema",
		// Xsi:      "http://www.w3.org/2001/XMLSchema-instance",
		// XsiType:  "bookstoreType",
		// XsdValue: "http://www.w3.org/2001/XMLSchema",
		Books: []Book{
			{
				Genre:  "Fiction",
				Title:  "The Catcher in the Rye",
				Author: Author{FirstName: "J.D.", LastName: "Salinger"},
				Price:  8.99,
			},
			{
				Genre:  "Fiction",
				Title:  "1984",
				Author: Author{FirstName: "George", LastName: "Orwell"},
				Price:  7.99,
			},
		},
	}

	enc := xml.NewEncoder(os.Stdout)
	enc.Indent("", "    ")
	if err := enc.Encode(bookstore); err != nil {
		panic(err)
	}
}
