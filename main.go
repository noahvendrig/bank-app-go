package main

// import F

func main() {
	server := NewAPIServer(":3000")
	server.Run()
}
