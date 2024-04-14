# syntax=docker/dockerfile:1-experimental

FROM archlinux

RUN pacman --noconfirm -Sy archlinux-keyring
RUN pacman --noconfirm -Sy iperf time

COPY bin bin

## When this container image is run, sleep
CMD ["sh", "-c", "sleep 100000000"]
