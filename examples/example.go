package main

import (
	"fmt"
	"github.com/Stebalien/go-json-doc"
)

type Address struct {
	Number                       int
	Street, City, State, Country string
	PostalCode                   int
}

func (a *Address) MarshalText() string {
	return fmt.Sprintf(
		"%d %s\n%s, %s, %s, %d",
		a.Number, a.Street,
		a.City, a.State, a.Country, a.PostalCode,
	)
}

type Person struct {
	Name       string
	Occupation string
	Age        int
	Address    Address
}

func main() {
	myGlossary := jsondoc.NewGlossary().
		WithName(new(Address), "street-address")

	description, err := myGlossary.Describe(new(Person))
	if err != nil {
		panic(err)
	}

	fmt.Println(description)
	// {
	//   "Name": "<string>",
	//   "Occupation": "<string>",
	//   "Age": "<int>",
	//   "Address": "<street-address>"
	// }
}
