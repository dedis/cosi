#!/bin/bash

## install.sh will fetch all dependencies of the project.
## If the branch from where the PR comes from starts with
## `refactor_`, it will checkout the same branch name
## in dedis/cosi.

# Temporarily overwrite the branch
BRANCH=$TRAVIS_PULL_REQUEST_BRANCH

pattern="refactor_*"
if [[ $BRANCH =~ $pattern ]]; then 
    echo "Using refactored branch $BRANCH - fetching cothority"
    repo=github.com/dedis/cothority
    go get -d $repo
    cd $GOPATH/src/$repo
    git checkout -f $BRANCH
    echo $(git rev-parse --abbrev-ref HEAD)
fi
echo "Using branch $BRANCH"
cd $TRAVIS_BUILD_DIR
