#!/bin/bash
#set -x
doSetupEnvironment() {
  # Set paths and environment variables
  export GOPROXY=https://repo.lab.pl.alcatel-lucent.com/gocenter
  export GONOSUMDB=*
  export PATH=${GOROOT}/bin:${PATH}
  echo "PATH is set to : "${PATH}
}

doGmMake() {
  echo ""
  echo "...."
  echo ".... Doing make for $1 in $(pwd)"
  echo "...."
  rm -f build/bin/*
  go env
  go mod tidy
  go mod vendor
  go build -ldflags "-s -w" -o build/bin/$1 $3
  if [ "$? " -ne 0 ]; then
    echo "build $1 failed"
    exit 1
  else
    echo "build $1 successfully"
  fi
  cp build/bin/$1  ${GMPS_TOP}_do/exec/rhlinux/debug/$2
  cp build/bin/$1  ${GMPS_TOP}_do/exec/rhlinux/release/$2

  if [ "$? " -ne 0 ]; then
    echo "doGmMake failed"
    exit 1
  else
    echo "doGmMake success"
  fi
}

doGmBuild() {
  doSetupEnvironment
  doGmMake $*
}

GIT_TOP=`git rev-parse --show-toplevel 2>/dev/null`
GIT_TOOL_TOP=/imsgit/Tools

GOROOT=${GIT_TOOL_TOP}/go1.20.3/go_64
export GOROOT

cd ${GIT_TOP}/cloud_native/debugassist/src/debugassist/debugassistServer
doGmBuild debugassist debugassist main.go
