#!/bin/bash

################################################################################
# Build tape binary in the root directory of tape project
################################################################################
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
  END_TO_END='true'
  CFG_FILE='config.yaml'

  SEED='2333'   # random seed (default 0)
  RATE='0'      # Creates tx rate, default 0 as unlimited (default 0)
  BURST='50000' # Burst size for Tape, should bigger than rate (default 1000)

  CONN_NUM='4'        # Number of grpc connections (default 16)
  CLIENT_PER_CONN='4' # Number of clients per grpc connection (default 16)
  ORDERER_CLIENT='5'  #TODO: Orderer clients (default 20)
  THREAD='1'          #TODO: Signature thread (default 1)
  ENDORSER_GROUP='1'  #TODO: Endorser groups

  TX_TYPE='put' # Transaction type ('put' or 'conflict')
  TX_NUM='2000' #TODO: Number of tx for shot (default 50000)
  TX_TIME='120' #TODO: Time of tx for shot (default 120s)

  # check crypto materials are ready
  ORGS_DIR="${NETWORK_PATH}/organizations"
  [[ -d ${ORGS_DIR} ]] || ln -sf ${ORGS_DIR}

  if [[ ${END_TO_END} == 'true' ]]; then
    ./tape -c "${CFG_FILE}" --seed "${SEED}" --rate "${RATE}" --burst "${BURST}" --num_of_conn "${CONN_NUM}" --client_per_conn "${CLIENT_PER_CONN}" --orderer_client "${ORDERER_CLIENT}" --thread "${THREAD}" --endorser_group "${ENDORSER_GROUP}" --txtype "${TX_TYPE}" --number "${TX_NUM}" --time "${TX_TIME}"
  else
    #TODO not end to end
    :
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
    run
    ;;
  'clean')
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
