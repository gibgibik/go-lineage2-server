FROM nvidia/cuda:12.9.1-cudnn-devel-ubuntu20.04
RUN apt-get update && apt-get -y install nano wget curl git unzip sudo
RUN wget https://go.dev/dl/go1.24.5.linux-amd64.tar.gz && tar -xf go1.24.5.linux-amd64.tar.gz -C /usr/local
ENV PATH="$PATH:/usr/local/go/bin"
RUN cd /tmp && mkdir testgocv && cd testgocv && go mod init testgocv && go get gocv.io/x/gocv@latest
RUN DEBIAN_FRONTEND=noninteractive  apt install -y  --no-install-recommends \
    build-essential \
    cmake \
    pkg-config \
    libgtk-3-dev \
    libavcodec-dev \
    libavformat-dev \
    libswscale-dev \
    libtbb2 \
    libtbb-dev \
    libjpeg-dev \
    libpng-dev \
    libtiff-dev \
    libdc1394-22-dev
RUN cd ~/go/pkg/mod/gocv.io/x/gocv\@v0.41.0 && make install_cuda