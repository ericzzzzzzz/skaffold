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

# This script creates a github issue if it hasn't been created when there
# are vulnerabilities found in the LTS image.

set -xeo pipefail


if [ -z "$_REPO" ]; then
  _REPO="https://github.com/ericzzzzzzz/kokoro-codelab-ericwork"
fi

OS_VULN_FILE=os_vuln.txt
IMAGES_TO_REPORT_FILE=os_vuln.txt

append() {
  echo -e $1 >> $2
}

find_issue() {
  label=$1
  issue=$(gh issue list --label "$label" --repo="$_REPO" --json number,title)
  echo "$issue"
}

create_issue() {
  title=$1
  label=$2
  image_tag=$3
  body="Vulnerabilities were found in skaffold binary, please fix the issues.
        Please make a patch release if the issues are in lts release.
        Vulnerabilities details: see [here]($image_tag)."

  gh label create --repo="$_REPO" "$label" -c "1D76DB" -d "skaffold binary has vulnerabilities" --force
  gh issue create --repo="$_REPO" --title="$title" --label="$label" --body="$body"
}

close_issue_as_fixed() {
   issue_num=$1
   tag=$2
   gh issue close "$issue_num" --repo="$_REPO" -c "Closing as the issue is fixed in $tag"
}

close_issue_tracked_in_another() {
   issue_num=$1
   new_issue_url=$2
   gh issue close "$issue_num" --repo="$_REPO" -c "Closing as the issue is tracked in $new_issue_url"
}

process_report_without_existing_issue() {
  title=$1
  label=$2
  vulnerable=$3
  image_tag=$4
  if [ "$vulnerable" == "true" ]; then
    new_issue_url=$(create_issue "$title" "$label" "$image_tag")
  fi
}

process_report_with_existing_issue() {
   issue=$1
   title=$2
   label=$3
   vulnerable=$4
   image_tag=$5

   issue_title=$(echo "$issue" | ggrep -oP '"title": *\K"[^"]*"' | head -n 1)
   issue_num=$(echo "$issue" | ggrep -oP 'number":\s*\K\d+' | head -n 1)

   if [ "$vulnerable" == "false" ]; then
      close_issue_as_fixed "$issue_num" "$image_tag"
   elif [ "$issue_title" != "$title" ]; then
      new_issue_url=$(create_issue "$title" "$label" "$image_tag")
      close_issue_tracked_in_another "$issue_num" "$new_issue_url"
   fi
}

while IFS= read -r line; do
    echo "Text read from file: $line"
    tag=$(echo "$line" | awk -F '[:]' '{print $2}')
    image_tag=$(echo "$line" | awk -F '[:]' '{print $1":"$2}')
    vulnerable=$(echo "$line" | awk -F '[:]' '{print $3}')
    label="bin-vul-${tag%.*}"
    title="skaffold vulnerabilities found in $tag binary"
    issue=$(find_issue "$label")
    if [ '[]' == "$issue" ]; then
      process_report_without_existing_issue "$title" "$label" "$vulnerable" "$image_tag"
    else
      process_report_with_existing_issue "$issue" "$title" "$label" "$vulnerable" "$image_tag"
    fi
done < os_vuln.txt

