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
  _REPO="GoogleContainerTools/skaffold"
fi

TITLE_OS="LTS image has OS vulnerability!"
OS_VULN_FILE=os_vuln.txt
IMAGES_TO_REPORT_FILE=/workspace/images_to_report.txt

find_issue() {
  label=$1
  issue=$(gh issue list --label "$label" --repo="$_REPO" --json number,title)
  echo "$issue"
}

while IFS= read -r line; do
    echo "Text read from file: $line"
    image_tag=$(echo "$line" | awk -F '[:]' '{print $2}')
    vulnerable=$(echo "$line" | awk -F '[:]' '{print $3}')
    label="bin-vul-${image_tag%.*}"
    title="skaffold vulnerabilities found in $image_tag binary"
    issue=$(find_issue "$label")
    if [ '[]' == "$issue" ]; then
      if [ "$vulnerable" == "true" ]; then
        echo "creating new issue"
      fi
    else
      issue_title=$(echo "$issue" | ggrep -oP '"title": *\K"[^"]*"' | head -n 1)
      issue_num=$(echo "$issue" | ggrep -oP 'number":\s*\K\d+' | head -n 1)
      if [ "$issue_title" != "$title" ]; then
        echo "Closing as $issue_num, $image_tag, $vulnerable"
        if [ "$vulnerable" == "true" ]; then
          echo "Creating...."
        fi
      else
        echo "Checking date and reminding again"
      fi
    fi

done < os_vuln.txt

append() {
  echo -e $1 >> $2
}

#
#check_existing_issue() {
#  query=$1
#  label=$2
#  # Returns the open issues. There should be only one issue opened at a time.
#  issue_num=$(gh issue list --search "$query" --label "$label" --repo="$_REPO" --json number | ggrep -oP 'number":\s*\K\d+' | head -n 1)
#
#  if [ "$issue_num" ]; then
#    echo >&2 "There is already an issue opened for the detected vulnerabilities in the LTS images." && echo "$issue_num"
#  else
#    echo "-1"
#  fi
#}

#
#init_body_file(){
# append "Please patch the below images with instructions mentioned [here](https://docs.google.com/document/d/1gYJVoBCZiRzUTQs_-wKsfhHdskiMtJtWWQyI-t0mhC8/edit?resourcekey=0-NdLapTumfpzxH_bri0fLZQ#heading=h.p4mphzyz8m7y).\n" "$IMAGES_TO_REPORT_FILE"
#
#
#}
#

#
#create_issue() {
#  title="$1"
#  body_file="$2"
#  label="$3"
#  # label with minor version bin-vul-v2.0, bin-vul-v1.37
#  gh label create label -c "1D76DB" -d "Skaffold binary has vulnerabilities." --force
#  gh issue create --title="${title}" --label="${label}" --body-file="$body_file" --repo="$_REPO"
#}
#
#update_issue() {
#  num="$1"
#  body_file="$2"
#  gh issue edit "$num" --body-file="$body_file" --repo="$_REPO"
#}
#
#gh auth login --with-token <token.txt
#issue_num=$(check_existing_issue "$_OS_VULN_LABEL")
#
#init_body_file
#if [ "$issue_num" -eq "-1" ]; then
#  echo "Creating an issue..."
#  create_issue "$TITLE_OS" "$IMAGES_TO_REPORT_FILE" "$_OS_VULN_LABEL"
#else
#  echo "Updating issue: #""$issue_num"
#  update_issue "$issue_num" "$IMAGES_TO_REPORT_FILE"
#fi
