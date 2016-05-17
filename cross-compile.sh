#!/usr/bin/env bash
# Compiles binaries for MacOS and Linux 64-bit versions
# Needs the chose_version.sh script
# Syntax:
# ./cross-compile.sh version [nocompile]
# if nocompile is given, only the tar is done.

if [ ! "$1" ]; then
  echo Please give a version-number
  exit
fi
VERSION=$1
APPS="cosi"
BUILD=cosi_bin

compile(){
    rm -rf $BUILD
    mkdir $BUILD
    for APP in $@; do
        for GOOS in linux darwin; do
          for GOARCH in amd64; do
            echo "Compiling $APP $GOOS / $GOARCH"
            export GOOS GOARCH
            go build -o $BUILD/$APP-$GOOS-$GOARCH .
          done
        done
    done
}

if [ ! "$2" ]; then
  go build
  echo "Cross-compiling for platforms and cpus"
  compile $APPS
fi

for a in $APPS; do
    cp -v chose_version.sh $BUILD/$a
done
cp dedis_group.toml $BUILD
cp README.md $BUILD
TAR=conode-$VERSION.tar.gz

echo "Creating $TAR"
tar cf $TAR -C $BUILD .

git tag $VERSION
git push origin $VERSION
