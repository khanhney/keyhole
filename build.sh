#! /bin/bash
# Copyright 2018 Kuei-chun Chen. All rights reserved.

# dep init

DEP=`which dep`

if [ "$DEP" == "" ]; then
    echo "dep command not found"
    exit
fi

if [ -d vendor ]; then
    UPDATE="-update"
fi

$DEP ensure $UPDATE
mkdir -p build
export ver="2.3.4"
export version="v${ver}-$(date "+%Y%m%d")"
env GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.version=$version" -o build/keyhole-osx-x64 keyhole.go
env GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=$version" -o build/keyhole-linux-x64 keyhole.go
env GOOS=windows GOARCH=amd64 go build -ldflags "-X main.version=$version" -o build/keyhole-win-x64.exe keyhole.go

if [ "$1" == "docker" ]; then
    docker build -t simagix/keyhole:latest -t simagix/keyhole:${ver} .
    docker rmi -f $(docker images -f "dangling=true" -q)
fi
#go build -o build/keyhole-osx-x64 keyhole.go
