#!/bin/bash

aws ecr create-repository --repository-name cpuperf --image-scanning-configuration scanOnPush=true
