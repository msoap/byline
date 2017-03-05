test:
	go test -cover

test-race:
	go test -cover -race

lint:
	golint .
	go vet .
	errcheck .

benchmark:
	go test -benchtime 5s -benchmem -bench .

gometalinter:
	gometalinter --vendor --cyclo-over=20 --line-length=150 --dupl-threshold=150 --min-occurrences=2 --enable=misspell --deadline=10m ./...

update-from-github:
	go get -u github.com/msoap/byline
