.PHONY: demo example

demo:
	go build -o poc ./proof-of-concept

colours:
	go build -o colours ./example/colours.go
