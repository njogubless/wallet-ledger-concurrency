FROM golang:1.23-bookworm

WORKDIR /app

# Starter code the agent must fix — NOT the solution.
COPY starter/ /app/

RUN go build ./... 2>&1 | tee /tmp/build.log || true
