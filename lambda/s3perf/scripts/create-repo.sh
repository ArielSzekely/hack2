#!/bin/bash

aws ecr create-repository --repository-name s3perf --image-scanning-configuration scanOnPush=true
