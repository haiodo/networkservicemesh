#!/usr/bin/env bash
FILES=$(git ls-files '*.yaml' '*.yml')
echo "Files to LINT ${FILES}"
yamllint -c .yamllint.yml $(git ls-files '*.yaml' '*.yml')