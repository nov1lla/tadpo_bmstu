FROM golang:1.20-bullseye

RUN apt-get update \
  && apt-get install -y --no-install-recommends \
    git \
    curl \
    tcpdump \
    postgresql-client \
    nodejs \
    npm \
  && rm -rf /var/lib/apt/lists/*

RUN npm install -g allure-commandline
RUN go install github.com/jstemmer/go-junit-report/v2@latest

WORKDIR /workspace
COPY docker/tests/entrypoint.sh /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]
