install_dev:
	for r in crypto cothority; do
		repo=github.com/dedis/$repo
		go get $repo
		cd $GOPATH/src/$repo
		git checkout development
	done
	cd $GOPATH/src/github.com/dedis/cosi
	go get -t ./...

test_fmt:
	files=$( go fmt ./... )
	if [ -n "$files" ]; then
		echo "Files not properly formatted: $files"
		exit 1
	fi

test_lint:
	go get -u github.com/golang/lint/golint
	failOn="should have comment or be unexported\| by other packages, and that stutters; consider calling this"
	lintfiles=$( golint ./... | grep "$failOn" )
	if [ -n "$lintfiles" ]; then
		echo "Lint errors: $lintfiles"
		exit 1
	fi

test: test_fmt test_lint
	go test -race -p=1 ./...
