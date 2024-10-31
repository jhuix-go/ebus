#!/bin/sh

SCRIPTS_PATH=$(cd "`dirname "$0"`"; pwd)
echo "Please select build platform OS:"

select OS in "Linux" "Windows" "Exit"; do
  case $OS in
    "Linux")
      export GOOS=linux
      OS_TYPE=linux
      EXTNAME=
      break
      ;;
    "Windows")
      export GOOS=windows
      OS_TYPE=win
      EXTNAME=.exe
      break
      ;;
    "Exit")
      exit
      ;;
    *) echo "Please select valid option" ;;
  esac
done

echo "build $OS platform version ..."
export GO111MODULE=on
CURR_PATH=${SCRIPTS_PATH%%/go/*}
if [ "$CURR_PATH" != "" ]; then
  export GOPATH=$CURR_PATH/go
fi
export GOMODCACHE=$GOPATH/pkg/mod
export GOARCH=amd64

cd ..
BUILD_TIME=$(date +%F\ %T)
COMMIT_HASH=$(git rev-parse --short HEAD)
VERSION=$(go version)
IFS=' ' read -r _ _ GO_VERSION _ <<< "$VERSION"
LDFLAGS="-s -w -X 'main.GoVersion=$GO_VERSION' -X 'main.BuildTime=$BUILD_TIME' -X 'main.CommitHash=$COMMIT_HASH'"

mkdir -p dist/$OS_TYPE/bin
mkdir -p dist/$OS_TYPE/conf

echo "build ebus service..."
go build -ldflags "$LDFLAGS" -o "dist/$OS_TYPE/bin/ebus$EXTNAME" && cp conf/ebus.toml dist/$OS_TYPE/conf/
cd $SCRIPTS_PATH
