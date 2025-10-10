test:
	rm -f test/*
	mkdir -p test
	go test ./... -coverprofile=test/coverage.out
	go tool cover -func=test/coverage.out
	go tool cover -html=test/coverage.out -o test/coverage.html
	open test/coverage.html
