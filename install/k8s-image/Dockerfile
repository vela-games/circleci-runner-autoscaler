FROM ubuntu:jammy

USER root

ENV TZ UTC

RUN useradd --create-home --shell /bin/bash circleci

RUN apt update && \
    apt install -y curl gzip sudo wget apt-transport-https unzip jq tzdata ca-certificates && \
    rm -rf /var/lib/apt/lists/* && \
    ln -fs /usr/share/zoneinfo/${TZ} /etc/localtime \
    && dpkg-reconfigure tzdata

RUN mkdir -p /opt/circleci/workdir
RUN chown -R circleci:circleci /opt/circleci

RUN echo "circleci ALL=(ALL) NOPASSWD: ALL" > /etc/sudoers.d/circleci
RUN echo "DEBIAN_FRONTEND=noninteractive" > /etc/environment

USER circleci

COPY --chown=circleci:circleci init-launch-agent.sh /opt/circleci/init-launch-agent.sh
RUN /bin/bash /opt/circleci/init-launch-agent.sh

COPY --chown=circleci:circleci start.sh /opt/circleci/start.sh
RUN chmod +x /opt/circleci/start.sh

ENTRYPOINT ["/opt/circleci/start.sh"]
