# ENERGY NETWORK UBUNTU
# BUILD IT with the script 'scrits/build-energy-network-ubuntu-image.sh'

ARG UBUNTU_VERSION=20.04


FROM ubuntu:${UBUNTU_VERSION}

RUN apt update
RUN apt install openjdk-13-jre -y
RUN apt install openjdk-13-jdk -y
ENV JAVA_HOME=$ARCH_JAVA_HOME
ENV PATH=$PATH:$JAVA_HOME/bin

ENV MAVEN_VERSION=3.6.3-1
ENV GIT_VERSION=1:2.25.1-1ubuntu3.1

RUN apt install maven="${MAVEN_VERSION}" -y

RUN apt install git="${GIT_VERSION}" -y

RUN git config --global core.autocrlf false && git config --global core.longpaths true

WORKDIR /install-patch/
RUN mkdir -p patches && mkdir -p scripts
COPY patches /install-patch/patches
COPY scripts /install-patch/scripts

RUN ./scripts/install-java-dependencies.sh
WORKDIR /
RUN rm -r /install-patch