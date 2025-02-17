# syntax=docker/dockerfile:1-experimental

FROM golang:alpine AS base

WORKDIR /app

RUN apk add --no-cache gcompat musl-dev strace

VOLUME /data
