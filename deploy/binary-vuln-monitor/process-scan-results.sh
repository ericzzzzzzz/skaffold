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

TITLE_OS="LTS image has OS vulnerability!"
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
  gh label create --repo="$_REPO" "$label" -c "1D76DB" -d "skaffold binary has vulnerabilities" --force
  gh issue create --repo="$_REPO" --title="$title" --label="$label" --body="message--"
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
  if [ "$vulnerable" == "true" ]; then
    echo "creating new issue title: $title, label: $label"
    new_issue_url=$(create_issue "$title" "$label")
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
    # we have a new version different from the vulnerable one mentioned in the issue.
    if [ "$issue_title" != "$title" ]; then
      if [ "$vulnerable" == "true" ]; then
        new_issue_url=$(create_issue "$title" "$label")
        close_issue_tracked_in_another "$issue_num" "$new_issue_url"
        echo "Closing as to be tracked in the new issue. $issue_num, $tag, $vulnerable, $new_issue_url"
      else
        close_issue_as_fixed "$issue_num" "$image_tag"
      fi
    else
      # This can edge binary, as we always use the same issue that for that. Also, it is possible to occur for two attempts scanning get different results if
      # scanner database gets updated, e.g. fix false positives
      if [ "$vulnerable" == "false" ]; then
        close_issue_as_fixed "$issue_num" "$image_tag"
      fi
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
      process_report_without_existing_issue "$title" "$label" "$vulnerable"
    else
      process_report_without_existing_issue "$issue" "$title" "$label" "$vulnerable" "$image_tag"
    fi
done < os_vuln.txt

