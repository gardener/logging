# Copyright (c) 2018 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

#############      builder       #############
FROM golang:1.11.4 AS builder

WORKDIR /go/src/github.com/gardener/logging
COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go install ./curator-es/extra-deleter/...

############# extra-deleter & curator #############

FROM python:2.7-slim

COPY --from=builder /go/bin/extra-deleter /bin
RUN pip install elasticsearch-curator==5.6.0

ENTRYPOINT [ "/usr/local/bin/curator" ]
