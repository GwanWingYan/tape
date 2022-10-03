#!/bin/bash

################################################################################
# Build tape binary in the root directory of tape project
################################################################################
function build_binary() {
  if [[ -f ./tape ]]; then
    echo "Already have the tape binary built."
    return 0
  fi

  go mod tidy && go build ./cmd/tape
  if [[ $? -ne 0 ]]; then
    echo "Fail to build tape binary."
    exit 1
  fi
}

################################################################################
# Build tape docker image
################################################################################
function build_docker() {
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

################################################################################
# Remove the binary and docker container
################################################################################
function clean() {
  rm -f ./tape 2>/dev/null
  docker image rm tape 2>/dev/null
}

################################################################################
# Run end to end test
################################################################################
function run() {
  local RESULT_FILE="$1"

  # import test options
  source ./test.env

  # check crypto materials are ready
  ORGS_DIR="${NETWORK_PATH}/organizations"
  [[ -d ${ORGS_DIR} ]] || ln -sf ${ORGS_DIR}

  if [[ ${END_TO_END} == 'true' ]]; then
    ./tape -c "${CFG_FILE}" --txtype "${TX_TYPE}" --number "${TX_NUM}" --time "${TX_TIME}" --endorser_group "${ENDORSER_GROUP}" --seed "${SEED}" --rate "${RATE}" --burst "${BURST}" --orderer_client "${ORDERER_CLIENT}" --num_of_conn "${CONN_NUM}" --client_per_conn "${CLIENT_PER_CONN}" --thread "${THREAD}" > "${RESULT_FILE}"
  else
    ./tape --no-e2e -c "${CFG_FILE}" --txtype "${TX_TYPE}" --number "${TX_NUM}" --time "${TX_TIME}" --endorser_group "${ENDORSER_GROUP}" --seed "${SEED}" --rate "${RATE}" --burst "${BURST}" --orderer_client "${ORDERER_CLIENT}" --num_of_conn "${CONN_NUM}" --client_per_conn "${CLIENT_PER_CONN}" --thread "${THREAD}" > "${RESULT_FILE}"
  fi
}

function main() {
  set -x
  case $1 in
    'build')
      shift
      if [[ -z $1 ]]; then
        build_binary
        build_docker
      elif [[ $1 == 'binary' ]]; then
        build_binary
      elif [[ $1 == 'docker' ]]; then
        build_docker
      else
        echo "Invalid build option"
      fi
      ;;
    'run')
      shift
      run "$1"
      ;;
    'clean')
      shift
      clean
      ;;
    *)
      echo 'Invalid option'
      exit 1
      ;;
  esac
  set +x
}

main $@