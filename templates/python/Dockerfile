ARG WATCHDOG_IMAGE_REPO=ghcr.io/openfaas/of-watchdog
ARG WATCHDOG_IMAGE_TAG=0.10.6
ARG PYTHON_IMAGE_REPO=python
ARG PYTHON_IMAGE_TAG=3.11-slim-bullseye


FROM --platform=${TARGETPLATFORM:-linux/amd64} ${WATCHDOG_IMAGE_REPO}:${WATCHDOG_IMAGE_TAG} as watchdog
FROM --platform=${TARGETPLATFORM:-linux/amd64} ${PYTHON_IMAGE_REPO}:${PYTHON_IMAGE_TAG} as build

COPY --from=watchdog /fwatchdog /usr/bin/fwatchdog
RUN chmod +x /usr/bin/fwatchdog

# Alternatively use ADD https:// (which will not be cached by Docker builder)
ARG ADDITIONAL_PACKAGE=""
RUN echo "ADDITIONAL_PACKAGE=${ADDITIONAL_PACKAGE}"
RUN apt-get -qy update \
    && apt-get -qy install ${ADDITIONAL_PACKAGE} \
    && rm -rf /var/lib/apt/lists/*

# Add non root user
RUN addgroup --system app && adduser app --system --ingroup app
RUN chown app /home/app

USER app

ENV PATH=$PATH:/home/app/.local/bin

WORKDIR /home/app/

ARG PROXY=""

COPY --chown=app:app main.py           .
COPY --chown=app:app requirements.txt   .
USER root
RUN pip install --no-cache-dir --proxy=${PROXY} -r requirements.txt
USER app

RUN mkdir -p function
RUN touch ./function/__init__.py
WORKDIR /home/app/function/
COPY --chown=app:app function/requirements.txt	.
RUN pip install --no-cache-dir --proxy=${PROXY} --user -r requirements.txt

USER root
COPY --chown=app:app function/   .

FROM build as test

ARG TEST_COMMAND=tox
ARG TEST_ENABLED=false

RUN [ "$TEST_ENABLED" = "false" ] && echo "skipping tests" || eval "$TEST_COMMAND"


FROM build as ship
WORKDIR /home/app/

USER app

# Set up of-watchdog for HTTP mode
ENV fprocess="python main.py"
ENV cgi_headers="true"
ENV mode="http"
ENV upstream_url="http://127.0.0.1:5000"
ENV ready_path="/_/ready"
ENV exec_timeout="0"
ENV read_timeout="3600"
ENV write_timeout="3600"
ENV healthcheck_interval="30"

HEALTHCHECK --interval=5s CMD [ -e /tmp/.lock ] || exit 1

CMD ["fwatchdog"]
