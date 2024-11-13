mac:
	rm -rf bin
	GOOS=darwin GOARCH=arm64 go build -o bin/darwin-arm64/viam-hough-transform

linux:
	rm -rf bin
	GOOS=linux GOARCH=arm64 go build -o bin/linux-arm64/viam-hough-transform