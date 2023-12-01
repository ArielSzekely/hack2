#!/bin/bash

aws ecr create-repository --repository-name lambda-start-latency --image-scanning-configuration scanOnPush=true
