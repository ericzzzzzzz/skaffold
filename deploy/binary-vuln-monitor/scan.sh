#!/bin/bash
# Copyright 2021 The Skaffold Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This script scans the vulnerability report that is generated by Container Analysis.

set -xeo pipefail
# Variables that will be substituted in cloudbuild.yaml.
if [ -z "$_TAG_FILTER" ]; then
  _TAG_FILTER="v.*-lts|^edge$"
fi
# us-east1-docker.pkg.dev/ericz-skaffold/eric-testing/skaffold
if [ -z "$_BASE_IMAGE" ] ; then
  _BASE_IMAGE="gcr.io/k8s-skaffold/skaffold"
fi
# If changed, also change the same variable in report.sh.
OS_VULN_FILE=os_vuln.txt

append() {
  printf "%s\n" $1 >>$2
}

check_vulnerability(){
  base_image=$1
  tags_filter=$2
  result_file=$3
  tags=$4
  tags_filter=""

  if [ -z "$tags" ]; then
    targeted_base_tags="$(gcloud container images list-tags "$base_image" --filter="timestamp.datetime > -P1Y AND tags~v.*\.1-lts" --format='value(tags)')"
    for line in $targeted_base_tags; do
      IFS=',' read -ra t <<< "${line}"
      replacement="\."
      t[0]="${t[0]//./$replacement}"
      tags_filter+="${t[0]/1-lts/.*-lts}|"
    done
    tags_filter+="^edge$"
    # get the latest patches tags for lts images. gcloud will return extra tags if an image has multiple tags and we only want tags specified in the filter, so use grep to further filter the result.
    tags=$(gcloud container images list-tags "$base_image" --filter="tags~$tags_filter" --format='value(tags)' | sort -nr | awk -F'[:.]' '$1$2!=p&&p=$1$2' | grep -Po "$tags_filter|edge")
  fi

  for tagsByComma in $tags; do
    IFS="," read -ra tagArr <<< "${tagsByComma}"
    image=$base_image:${tagArr[0]}
    echo "Checking vulnerabilities of image:" "$image"
    gcloud beta container images describe "$image"  --show-package-vulnerability \
     | if grep -e "effectiveSeverity: HIGH" -e "effectiveSeverity: CRITICAL";
       then
         append "$base_image:$tagsByComma:true" "$result_file";
       else
         append "$base_image:$tagsByComma:false" "$result_file"
       fi
  done
}

# Main
# Scans the LTS images
check_vulnerability $_BASE_IMAGE "$_TAG_FILTER"  "$OS_VULN_FILE" "$_TAGS"
