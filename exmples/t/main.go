package main

import "fmt"

func main() {
	x := 10
	defer print(x)
	x = 11
}

func print(x int) {
	fmt.Println(x)
}
