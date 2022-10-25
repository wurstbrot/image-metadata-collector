#!/bin/bash
#shellcheck disable=SC2155

wait_for_pods_ready () {
  local -r name="${1}"; shift
  local -r namespace="${1}"; shift
  local -r count="${1}"; shift
  local -r sleep="${1}"; shift
  local -r max_attempts="${1}"
  local attempt_num=0

  until [[ $(kubectl -n "${namespace}" get pods -o json | jq '.items | length') -ge "${count}" ]]
  do
    if [[ $(( attempt_num++ )) -ge "${max_attempts}" ]]
    then
      echo "max_attempts ${max_attempts} reached, aborting"
      debug_pods_in_namespace "${namespace}"
      exit 1
    fi
    echo "waiting for ${name} to be created"
    sleep "${sleep}"
  done
  until [[ $(kubectl -n "${namespace}" get pods -o json | jq '.items[].status.phase' | grep -c Pending) -eq "0" ]]
  do
    if [[ $(( attempt_num++ )) -ge "${max_attempts}" ]]
    then
      echo "max_attempts ${max_attempts} reached, aborting"
      debug_pods_in_namespace "${namespace}"
      exit 1
    fi
    echo "waiting for ${name} to not be Pending"
    sleep "${sleep}"
  done
  until [[ $(kubectl -n "${namespace}" get pods -o json | jq '.items[].status.conditions[].status=="True"' | grep -c false) -eq "0" ]]
  do
    if [[ $(( attempt_num++ )) -ge "${max_attempts}" ]]
    then
      echo "max_attempts ${max_attempts} reached, aborting"
      debug_pods_in_namespace "${namespace}"
      exit 1
    fi
    echo "waiting for ${name} to be up"
    sleep "${sleep}"
  done
}

wait_for_pods_completed () {
  local -r name="${1}"; shift
  local -r namespace="${1}"; shift
  local -r count="${1}"; shift
  local -r sleep="${1}"; shift
  local -r max_attempts="${1}"
  local -r attempt_num=0

  until [[ $(kubectl get pods -n ${namespace} | grep -c Running) -eq ${count} ]]
  do
    if [[ $(( attempt_num++ )) -ge "${max_attempts}" ]]
    then
      echo "max_attempts ${max_attempts} reached, aborting"
      debug_pods_in_namespace "${namespace}"
      exit 1
    fi
    echo "waiting for ${name} to be created"
    sleep "${sleep}"
  done
}

wait_for_http()  {
  local -r url="${1}";
  local -r max_attempts="120";
  local attempt_num=0
  sleep=5

  until curl ${url}; do
    if [[ $(( attempt_num++ )) -ge "${max_attempts}" ]]
    then
      echo "max_attempts ${max_attempts} reached, aborting"
      exit 1
    fi
    sleep ${sleep}
  done
}

debug_pods_in_namespace() {
  local -r namespace="${1}"
  kubectl get pods -A
  for pod in $(kubectl get pods -n ${namespace} | grep -v NAME  | awk '{print $1}'); do
    echo "######################### ${pod}:"
    kubectl get pod -n ${namespace} ${pod} -o yaml
    kubectl logs -n ${namespace} ${pod}
    echo "#########################"
  done
}


export BRANCH=$(git rev-parse --abbrev-ref HEAD)
export MAJOR="2"
export MINOR="0"
export PATCH="${GITHUB_RUN_NUMBER}"
export VERSION="${MAJOR}.${MINOR}.${PATCH}"
if [ "${BRANCH}" != "master" ] && [ "${BRANCH}" != "head" ]; then
  export MAJOR=$(echo ${BRANCH} | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9._-]//g')
  export PATCH=""
  if [ "${GITHUB_RUN_NUMBER}" != "" ]; then
    echo "Detected GITHUB_RUN_NUMBER"
    export MINOR="${GITHUB_RUN_NUMBER}"
    if [ "${GITHUB_HEAD_REF}" != "" ]; then
      echo "Detected GITHUB_HEAD_REF"
      export MAJOR=$(echo ${GITHUB_HEAD_REF} | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9._-]//g')
    fi
  else
    export MINOR=""
  fi
  export VERSION="${MAJOR}.${MINOR}.${PATCH}"
fi
VERSION=$(echo ${VERSION} | sed 's#\.$##')
VERSION=$(echo ${VERSION} | sed 's#\.$##') # to heal two .
echo "VERSION: ${VERSION}"

export TMP_FOLDER=/tmp
compareDownloadedFileWithExpected() {
  filename=$1
  echo "Checking file ./expectedFiles/${filename} ${TMP_FOLDER}/${filename}"
  filenameCreateFile=${filename}
  if [ "$(echo "${filename}" | grep -c json)" -eq 1 ]; then
    echo "Normalizing json (${TMP_FOLDER}/1xx)"
    cat ${TMP_FOLDER}/${filename}  | jq '.[] | del(.image_id)' | sed 's/[a-z]:.*//' | sed 's/[0-9]:.*//' > ${TMP_FOLDER}/1xx
    filenameCreateFile="1xx"
  fi
  diff ./expectedFiles/${filename} ${TMP_FOLDER}/${filenameCreateFile}
  if [ $? -ne 0 ]; then
    echo "File ${filename} is not as expected"
    echo "#########################./expectedFiles/${filename}:"
    cat ./expectedFiles/${filename}

    echo "#########################${TMP_FOLDER}/${filename}:"
    cat ${TMP_FOLDER}/${filename}
    exit 1
  fi
  echo "->Files are the same"
}
