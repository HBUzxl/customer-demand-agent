# customer-demand-agent —— 生产镜像（P9 Docker 约定）
# 构建：docker build -t cda .
# 运行：docker run -p 8080:8080 -v cda-data:/var/lib/cda cda
#   状态全在卷 /var/lib/cda（wiki + history.db）；镜像无状态。
#   首次启动自动从内置种子播种 wiki；config.json 经卷挂载或环境定制。
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /cda-agent ./cmd/agent

FROM alpine:3.20
WORKDIR /app
COPY --from=build /cda-agent /app/cda-agent
COPY wiki /app/wiki
ENV CDA_DATA_DIR=/var/lib/cda
VOLUME /var/lib/cda
EXPOSE 8080
ENTRYPOINT ["/app/cda-agent"]
CMD ["--config", "/app/config.json"]
