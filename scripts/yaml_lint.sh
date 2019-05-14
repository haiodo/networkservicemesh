#!/usr/bin/env bash
yamllint -c .yamllint.yml $(git ls-files '*.yaml' '*.yml')