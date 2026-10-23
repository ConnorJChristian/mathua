# The Go builder comes first: it also merges the enabled modules' concepts,
# which the web build below bundles.
#
# Node pinned to the 2026-10-21 security release (CVE-2026-111497, CVE-2026-106069,
# CVE-2026-105483, CVE-2026-105482, and the medium fixes). The major-only tag
# `node:24-bookworm` is whatever the builder has cached, so a rebuild could ship
# the previous, vulnerable patch. Node is build-time only; the runtime image
# below does not contain it.
FROM golang:1.27-bookworm AS builder

RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc libc6-dev libsqlite3-dev && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 go build -o /mathua ./cmd/mathua

# Modules (internal/modules): the web app bundles the concept list, so the
# enabled modules, your modules-local/ included, are merged here and handed to
# the web build below. modules.local.json is optional; an empty one is made
# when it is absent so the runtime stage can always copy it.
RUN test -f modules.local.json || echo '{}' > modules.local.json \
    && go run ./scripts/merge_concepts -local

FROM node:24.22.1-bookworm AS webbuilder

WORKDIR /app
COPY web/next-app/package.json web/next-app/package-lock.json ./web/next-app/
RUN cd web/next-app && npm ci

COPY web/next-app ./web/next-app
# The merged concept and track lists from the builder stage (your modules
# included), and the raw corpus and modules the graph layout reads.
COPY --from=builder /app/web/next-app/data/concepts.json ./web/next-app/data/concepts.json
COPY --from=builder /app/web/next-app/data/tracks.json ./web/next-app/data/tracks.json
COPY --from=builder /app/modules.local.json ./modules.local.json
COPY data/concepts/ ./data/concepts/
COPY modules/ ./modules/
COPY modules-local/ ./modules-local/
ENV MATHUA_INCLUDE_LOCAL_MODULES=1
# Stamp a unique service-worker version per image build so every deploy
# migrates clients off older caches (see public/sw.js). Without this, a
# static SW version lets browsers keep serving stale hashed JS forever.
# The grep guard fails the build loudly if the stamp pattern drifts.
RUN STAMP="build-$(date -u +%Y%m%dT%H%M%SZ)" && \
    sed -i "s/^const VERSION = '[^']*'/const VERSION = 'v2-${STAMP}'/" ./web/next-app/public/sw.js && \
    grep -q "^const VERSION = 'v2-build-[0-9]\{8\}T" ./web/next-app/public/sw.js
# Single-binary: same origin => relative fetches. Empty API base bakes fetch('/api/...') 
# which works on any Fly hostname (mathua.fly.dev, mathua-xxx.fly.dev). For split Vercel+Fly,
# set build arg NEXT_PUBLIC_API_URL=https://mathua.fly.dev
ARG NEXT_PUBLIC_API_URL=""
ENV NEXT_PUBLIC_API_URL=$NEXT_PUBLIC_API_URL
# This image serves the export and the API from one origin, so /api/* is correct.
ENV NEXT_PUBLIC_SAME_ORIGIN_API=1
RUN cd web/next-app && npm run build

# Python track bank check, on the same base and pins as the runtime image.
FROM debian:bookworm-slim AS pybankcheck
RUN apt-get update && apt-get install -y --no-install-recommends \
    python3 python3-venv && rm -rf /var/lib/apt/lists/*
COPY grading/requirements-python.txt /check/grading/requirements-python.txt
RUN python3 -m venv /opt/venv \
    && /opt/venv/bin/pip install --no-cache-dir --disable-pip-version-check \
        -r /check/grading/requirements-python.txt
COPY grading/pyrunner.py /check/grading/pyrunner.py
COPY scripts/python_bank.py /check/scripts/python_bank.py
COPY modules/ /check/modules/
COPY modules-local/ /check/modules-local/
# Every authored bank, committed and local, is re-run against these pins.
RUN mkdir -p /tmp/mpl && MPLCONFIGDIR=/tmp/mpl /opt/venv/bin/python3 -c "import matplotlib.pyplot" \
    && cd /check && MPLCONFIGDIR=/tmp/mpl /opt/venv/bin/python3 scripts/python_bank.py check \
    && touch /bankcheck.ok

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    libsqlite3-0 ca-certificates curl python3 python3-venv && rm -rf /var/lib/apt/lists/*

# SymPy grading runtime. findSymPyService resolves ./grading/sympy_service.py
# from WORKDIR /app, and execs `python3`, so the venv must be first on PATH.
COPY grading/requirements.txt /app/grading/requirements.txt
COPY grading/requirements-python.txt /app/grading/requirements-python.txt
RUN python3 -m venv /opt/venv \
    && /opt/venv/bin/pip install --no-cache-dir --disable-pip-version-check \
        -r /app/grading/requirements.txt \
        -r /app/grading/requirements-python.txt
ENV PATH=/opt/venv/bin:$PATH

# Python track: Matplotlib builds its font cache on first import, which takes
# seconds and needs a writable folder. Learner code runs as `nobody` and may
# only write in its own temporary folder, so the cache is built here once and
# shared read-only. The import check runs as `nobody` because that is who
# imports these libraries at run time (internal/pytrack/sandbox_unix.go).
ENV MPLCONFIGDIR=/opt/mplconfig
RUN mkdir -p /opt/mplconfig \
    && MPLBACKEND=Agg python3 -c "import matplotlib.pyplot" \
    && chmod -R a+rX /opt/mplconfig /opt/venv \
    && runuser -u nobody -- env MPLCONFIGDIR=/opt/mplconfig MPLBACKEND=Agg \
        /opt/venv/bin/python3 -c "import numpy, pandas, sklearn, matplotlib.pyplot"

COPY --from=builder /mathua /mathua
COPY data/concepts/ /app/data/concepts/
COPY data/lessons/ /app/data/lessons/
COPY data/courses.json /app/data/courses.json
COPY data/destinations.json /app/data/destinations.json
# Modules: committed ones and your own (internal/modules). Concepts, question
# banks, courses and readings are read at startup; files/ holds your PDFs.
COPY modules/ /app/modules/
COPY modules-local/ /app/modules-local/
COPY --from=builder /app/modules.local.json /app/modules.local.json
COPY --from=webbuilder /app/web/next-app/out /app/web/next-app/out
COPY grading/sympy_service.py /app/grading/sympy_service.py
COPY grading/pyrunner.py /app/grading/pyrunner.py

# Build-time grading self-test: if SymPy cannot grade, the image does not build,
# so a broken grading runtime can never be deployed. `5^4` and `625` are
# equivalent but not string-equal, so this exercises the real equivalence path.
RUN printf '%s\n' '{"id":"1","expected":"5^4","answer":"625"}' \
      | python3 /app/grading/sympy_service.py \
      | grep -q '"correct": true'

# Build-time Python track self-test: every printed output stored in the
# question bank is recomputed with the libraries this image pins, every
# reference solution must pass its hidden tests and every starter must fail.
# It runs in a throwaway stage (pybankcheck, above) so the authoring sources stay out of
# the image; the final stage copies its marker, which forces it to run.
COPY --from=pybankcheck /bankcheck.ok /app/grading/bankcheck.ok

WORKDIR /app
ENV PORT=8080
ENV DATABASE_URL=/data/mathua.db

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD curl -fsS http://localhost:8080/api/health || exit 1

CMD ["/mathua", "-serve", "-port", "8080"]
