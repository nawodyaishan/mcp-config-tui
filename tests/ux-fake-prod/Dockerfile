FROM golang:1.24-bookworm

WORKDIR /repo

ENV NO_COLOR=1
ENV TERM=xterm-256color
ENV USYNC_TARGET_OS=linux

COPY . .

RUN go build -o /usr/local/bin/usync ./cmd/usync

ENTRYPOINT ["bash", "tests/ux-fake-prod/run-flow.sh"]
