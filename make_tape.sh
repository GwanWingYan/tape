#!/bin/bash

function build_binary() {
  if [[ -f ./tape || -f ./benchmark ]]; then
    echo "Already have the tape binary built."
    return 0
  fi

  go mod tidy && go build ./cmd/tape
  # go mod tidy && go build -o bench ./benchmark
  if [[ $? -ne 0 ]]; then
    echo "Fail to build tape binary."
    exit 1
  fi
}

function build_docker_image() {
  # To fetch from private Github repo, a user name and a PAT must be supplied
  GITHUB_USER=
  GITHUB_PAT=
  source ./github_credential

  if [[ ! -f ./Dockerfile ]]; then
    echo "No Dockerfile in current directory. Exit."
    exit 1
  fi

  docker build -f Dockerfile -t tape --build-arg GITHUB_USER=${GITHUB_USER} --build-arg GITHUB_PAT=${GITHUB_PAT} .
  if [[ $? -ne 0 ]]; then
    echo "Fail to build tape docker image."
    exit 1
  fi

  # remove intermediate images
  docker image prune --force --filter label=stage=builder
}

function clean() {
  rm -f ./tape 2>/dev/null
  docker image rm tape 2>/dev/null
}

function main() {
  case $1 in
  'build')
    build_binary
    build_docker_image
    ;;
  'clean')
    clean
    ;;
  *)
    echo 'Invalid option'
    exit 1
    ;;
  esac
}

main $@
