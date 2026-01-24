FROM golang:1.20-bullseye AS build

WORKDIR /src

COPY code/components/data/go.mod code/components/data/go.sum code/components/data/
COPY code/components/business/go.mod code/components/business/go.sum code/components/business/
COPY code/apps/webapp/go.mod code/apps/webapp/
COPY code/sdk/go.mod code/sdk/

RUN set -eu; \
    cd code/components/data && go mod download; \
    cd /src/code/components/business && go mod download; \
    cd /src/code/apps/webapp && go mod download

COPY . .
ARG BENCH_RUN_ID=default
LABEL bench.run_id=${BENCH_RUN_ID}

RUN mkdir -p /out/plugins /out/static
RUN cd code/components/data && \
    go build -tags plugin -buildmode=plugin -o /out/plugins/data.so ./cmd/plugin
RUN cd code/components/business && \
    go build -tags plugin -buildmode=plugin -o /out/plugins/business.so ./cmd/plugin
RUN cd code/apps/webapp && \
    go build -o /out/webapp ./cmd/webapp
RUN if [ -d code/components/webui/dist ]; then \
      cp -R code/components/webui/dist/* /out/static/; \
    else \
      echo '<html>benchmark</html>' > /out/static/index.html; \
    fi

FROM debian:bullseye-slim

WORKDIR /app
ARG BENCH_RUN_ID=default
LABEL bench.run_id=${BENCH_RUN_ID}
COPY --from=build /out/webapp /app/webapp
COPY --from=build /out/plugins /app/plugins
COPY --from=build /out/static /app/static

ENV WEBAPP_ADDR=:8080
ENV WEBAPP_STATIC_DIR=/app/static
ENV DATA_PLUGIN_PATH=/app/plugins/data.so
ENV BUSINESS_PLUGIN_PATH=/app/plugins/business.so

ENTRYPOINT ["/app/webapp"]
