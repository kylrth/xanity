FROM golang:1.20 as builder

ARG TARGETARCH

COPY . /usr/src/sanity
WORKDIR /usr/src/sanity
ENV GOOS=linux
ENV GOARCH=$TARGETARCH
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go get -v ./... && go build -o /sanity ./cmd/

# We'll download the arxiv-sanity-lite git repo and pip dependencies here and then copy over, to
# keep the runtime image small.
FROM python:3.10-slim as downloader

# system build dependencies
RUN --mount=type=cache,target=/var/cache/apt,sharing=locked \
    --mount=type=cache,target=/var/lib/apt,sharing=locked \
    apt-get update && \
    apt-get install -y \
        git \
        g++ \
        gcc \
        wget \
    && rm -rf /var/lib/apt/lists/*

# get the repository
ARG COMMIT=d7a303b
RUN git clone https://github.com/karpathy/arxiv-sanity-lite && \
    cd arxiv-sanity-lite && \
    git reset --hard $COMMIT && \
    sed -i "s|DATA_DIR = '."'*'"|DATA_DIR = '/data'|" aslite/db.py

# dependencies (TODO build sklearn before getting the repository by manually specifying the version
# in requirements.txt, so we don't have to wait so long every time)
RUN --mount=type=cache,target=/root/.cache/pip,sharing=locked \
    pip install --upgrade pip && \
    pip install --prefix="/install" \
        -r /arxiv-sanity-lite/requirements.txt \
        sendgrid

# get Tini
# https://github.com/krallin/tini/releases
ARG TINI_VERSION=0.19.0
ARG TINI_SHA256=93dcc18adc78c65a028a84799ecf8ad40c936fdfc5f2a57b1acda5a8117fa82c
RUN wget --quiet https://github.com/krallin/tini/releases/download/v${TINI_VERSION}/tini && \
    echo "${TINI_SHA256} *tini" | sha256sum -c - && \
    chmod +x tini

# runtime image
FROM python:3.10-slim

# system updates
RUN --mount=type=cache,target=/var/cache/apt,sharing=locked \
    --mount=type=cache,target=/var/lib/apt,sharing=locked \
    apt-get update && \
    apt-get upgrade -y && \
    apt-get install -y \
        curl \
        libgomp1 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /arxiv-sanity-lite

# copy runtime files and dependencies
COPY --from=downloader /install /usr/local
COPY --from=downloader /tini /usr/local/bin/tini
COPY --from=downloader /arxiv-sanity-lite/static/ ./static/
COPY --from=downloader /arxiv-sanity-lite/templates/ ./templates/
COPY --from=downloader /arxiv-sanity-lite/aslite/ ./aslite/
COPY --from=downloader /arxiv-sanity-lite/*.py ./
COPY --from=builder /sanity /sanity

# allow container to be run as non-root user
RUN chmod -R a+rw .

ENTRYPOINT ["tini", "--", "/sanity"]
CMD ["serve"]

EXPOSE 80/tcp
HEALTHCHECK --start-period=30s --interval=1m --timeout=5s \
    CMD curl -fSs http://localhost:80/about || exit 1
