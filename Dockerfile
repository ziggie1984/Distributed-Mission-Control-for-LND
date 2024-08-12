# Use an official Go runtime as a parent image.
FROM golang:1.22.2 AS build

# Install necessary tools and dependencies.
RUN apt-get update && \
    apt-get install -y \
        git \
        curl \
        build-essential \
        protobuf-compiler \
        make

# Create a directory for cloning the repository.
RUN mkdir /app

# Clone the Distributed Mission Control for LND repo into the /app directory.
RUN git clone https://github.com/ziggie1984/Distributed-Mission-Control-for-LND.git /app

# Run the installation scripts.
RUN /app/scripts/install_buf.sh && \
    /app/scripts/install_protobuf_plugins.sh

# Change current working directory.
WORKDIR /app

# Install Go modules.
RUN go mod download

# Build the EC Daemon.
RUN make build

# Install the EC Daemon.
RUN make install INSTALL_PATH="/go/bin"

# Add a non-root user for security.
RUN useradd -ms /bin/sh ecuser
USER ecuser

# Expose port 50050 for gRPC communication.
EXPOSE 50050

# Expose port 8081 for HTTP/1.1 REST communication.
EXPOSE 8081

# Expose port 6060 for pprof communication.
EXPOSE 6060

# Set the entrypoint to the EC Daemon.
ENTRYPOINT ["ec"]
