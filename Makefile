mac:
	rm -rf bin
	GOOS=darwin GOARCH=arm64 go build -o bin/hough-transform

linux:
	rm -rf bin
	GOOS=linux GOARCH=arm64 go build -o bin/hough-transform