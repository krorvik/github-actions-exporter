#!/bin/bash
#
# APIレスポンスの確認
#

set -u

for repo in $(echo "${GITHUB_REPOS}" | grep -o -E '[^,]+')
do
  now=$(date '+%Y-%m-%d_%H:%M:%S')
  output_dir="tmp/${repo}"
  mkdir -p "${output_dir}"
  echo "repo: ${repo}, output_dir: ${output_dir}"

  curl -s -H "Authorization: token ${GITHUB_TOKEN}" \
    "https://api.github.com/repos/${repo}/actions/runs" > "${output_dir}/${now}.runs.json"

  curl -s -H "Authorization: token ${GITHUB_TOKEN}" \
    "https://api.github.com/repos/${repo}/actions/workflows" > "${output_dir}/${now}.workflows.json"

  # ワークフロー毎の履歴
  for workflow_id in $(jq -r '.workflows[]|.id' "${output_dir}/${now}.workflows.json")
  do
    echo ${workflow_id}
    curl -s -H "Authorization: token ${GITHUB_TOKEN}" \
      "https://api.github.com/repos/${repo}/actions/workflows/${workflow_id}/runs" > "${output_dir}/${now}.workflow.${workflow_id}.json"
  done
done
