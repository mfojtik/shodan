#!/bin/bash

set -x
set -e

repository_fork_name="${1}"
repository_owner="${2}"
repository_name="${3}"
repository_branch="${4}"
go_module_name="${5}"
go_module_branch="${6}"
target_branch="${7}"

# make sure we don't stop on host verification
git config --global core.sshCommand 'ssh -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no'

echo "-> Bumping ${repository_owner}/${repository_name}@${repository_branch} dependency ${go_module_name}@${go_module_branch} ..."

# Clone the repository first
mkdir -p "/go/src/github.com/${repository_owner}"
pushd "/go/src/github.com/${repository_owner}"

git clone git@github.com:${repository_fork_name}/${repository_name}
popd

pushd "/go/src/github.com/${repository_owner}/${repository_name}"
git config --global user.name "shodan-bot"
git config --global user.email "shodan@mfojtik.io"

git remote add upstream git@github.com:${repository_owner}/${repository_name}
git fetch upstream
git checkout -b ${target_branch} upstream/${repository_branch}

go mod edit --require="${go_module_name}@${go_module_branch}"

go mod tidy -v
go mod vendor -v

go mod edit --fmt
go mod edit --print

git add --all
git commit -m "bump(*): revendoring ${go_module_name}/${go_module_branch}"
git push origin ${target_branch}

gh auth login --with-token < /root/.ssh/token
gh pr create --title "WIP: testing - ignore" --body "This is just a test" --base ${repository_branch} --head shodan-bot:${target_branch}

popd
