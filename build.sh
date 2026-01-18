# Windows 64-bit
GOOS=windows GOARCH=amd64 go build -o zenith-win.exe main.go

# macOS 64-bit
GOOS=darwin GOARCH=amd64 go build -o zenith-mac main.go

# Linux 64-bit
GOOS=linux GOARCH=amd64 go build -o zenith-linux main.go
