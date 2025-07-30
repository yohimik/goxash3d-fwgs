FROM debian:bookworm-slim AS engine

RUN dpkg --add-architecture i386
RUN apt update && apt upgrade -y && apt -y --no-install-recommends install aptitude
RUN aptitude -y --without-recommends install git ca-certificates build-essential gcc-multilib g++-multilib libbsd-dev:i386 libsdl2-dev:i386 libfreetype-dev:i386 libopus-dev:i386 libbz2-dev:i386 libvorbis-dev:i386 libopusfile-dev:i386 libogg-dev:i386

ENV PKG_CONFIG_PATH=/usr/lib/i386-linux-gnu/pkgconfig

WORKDIR /xash

COPY ./xash3d-fwgs .

RUN ./waf configure -T release -d --enable-lto --enable-openmp \
    && ./waf build

FROM golang:1.24 AS go

WORKDIR /go
COPY pkg pkg
COPY --from=engine /xash/build/engine/libxash.a pkg/libxash.a
COPY --from=engine /xash/build/public/libbuild_vcs.a pkg/libbuild_vcs.a
COPY --from=engine /xash/build/public/libpublic.a pkg/libpublic.a
COPY --from=engine /xash/build/3rdparty/libbacktrace/libbacktrace.a pkg/libbacktrace.a

COPY go.mod go.mod
COPY go.sum go.sum
COPY go.work go.work
COPY examples examples

WORKDIR /go/examples/webrtc-cs-i386

ENV GOARCH=386
ENV CC="gcc -m32 -D__i386__"
ENV CGO_CFLAGS="-fopenmp -m32"
ENV CGO_LDFLAGS="-fopenmp -m32"
RUN go mod download && go build .


FROM debian:bookworm-slim AS hlds

ARG hlds_build=8308
ARG hlds_url="https://github.com/DevilBoy-eXe/hlds/releases/download/$hlds_build/hlds_build_$hlds_build.zip"

RUN groupadd -r xash && useradd -r -g xash -m -d /opt/xash xash
RUN usermod -a -G games xash

RUN apt-get -y update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    unzip \
    && apt-get -y clean

USER xash
WORKDIR /opt/xash
SHELL ["/bin/bash", "-o", "pipefail", "-c"]

RUN mkdir -p /opt/xash/xashds

RUN curl -sLJO "$hlds_url" \
    && unzip "hlds_build_$hlds_build.zip" -d "/opt/xash/hlds_build_$hlds_build" \
    && cp -R "hlds_build_$hlds_build/hlds"/* xashds/ \
    && rm -rf "hlds_build_$hlds_build" "hlds_build_$hlds_build.zip"

# Fix warnings:
# couldn't exec listip.cfg
# couldn't exec banned.cfg
RUN touch /opt/xash/xashds/valve/listip.cfg
RUN touch /opt/xash/xashds/valve/banned.cfg

WORKDIR /opt/xash/xashds

# Copy default config
COPY configs/valve valve
COPY configs/cstrike cstrike

FROM debian:bookworm-slim AS final

ENV XASH3D_BASEDIR=/xashds

RUN dpkg --add-architecture i386
RUN apt-get update && apt-get install -y --no-install-recommends \
    libgcc-s1:i386 \
    libstdc++6:i386 \
    libgomp1:i386 \
    ca-certificates \
    openssl \
    && apt-get clean

RUN groupadd xashds && useradd -m -g xashds xashds
USER xashds
WORKDIR /xashds
ENV LD_LIBRARY_PATH=/xashds

COPY --from=hlds /opt/xash/xashds .
COPY --from=go /go/examples/webrtc-cs-i386/webrtc-cs ./xash
COPY --from=engine /xash/build/filesystem/filesystem_stdio.so ./filesystem_stdio.so
COPY --from=engine "/usr/lib/i386-linux-gnu/libstdc++.so.6" "./libstdc++.so.6"
COPY --from=engine "/usr/lib/i386-linux-gnu/libgcc_s.so.1" "./libgcc_s.so.1"
EXPOSE 27015/udp

# Start server
ENTRYPOINT ["./xash", "+ip", "0.0.0.0", "-port", "27015", "-game", "cstrike", "+maxplayers", "16"]

# Default start parameters
CMD ["+map de_dust2"]