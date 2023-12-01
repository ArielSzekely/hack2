#!/bin/bash

aws ecr create-repository --repository-name seccomp-allowlist --image-scanning-configuration scanOnPush=true
