echo "[sorting imports]"
goimports-reviser ./...

echo "[formatting code]"
gofumpt -w .

echo "[running go mod tidy]"
go mod tidy

echo "[compiling project]"
go build -o goupdate.exe cmd/goupdate/main.go

echo "[running golangci-lint]"
golangci-lint run
