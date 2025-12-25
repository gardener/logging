# Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.25-alpine AS builder

WORKDIR /workspace

COPY go.mod go.mod
COPY main.go main.go

RUN go mod download
RUN CGO_ENABLED=0 go build -a -o fetcher main.go

FROM gcr.io/distroless/static:nonroot

WORKDIR /

COPY --from=builder /workspace/fetcher .

USER 65532:65532

ENTRYPOINT ["/fetcher"]

