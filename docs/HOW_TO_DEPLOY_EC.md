# How to Deploy EC

This document explains how to deploy the EC Daemon using Docker, leveraging the
Dockerfile provided in the root directory of your project.

## Prerequisites

1. **Docker**: Ensure Docker is installed on your system. If not, follow the
instructions on the
[official Docker website](https://docs.docker.com/get-docker/).

## Building the Docker Image

1. **Navigate to the Project Directory**

   Open a terminal and navigate to the directory containing the `Dockerfile`:

   ```bash
   cd /path/to/your/project
   ```

2. **Build the Docker Image**

   Run the following command to build the Docker image from the Dockerfile:

   ```bash
   docker build -t ec-daemon .
   ```

   This command will build the Docker image and tag it as `ec-daemon`.

## Running the Docker Container

After building the Docker image, you can start the container using:

```bash
docker run --rm -p 50050:50050 -p 6060:6060 -p 8081:8081 --name ec-daemon-container ec-daemon
```

### Explanation

- `--rm`: Automatically remove the container when it exits.
- `-p 50050:50050`: Map port 50050 of the host to port 50050 of the container for gRPC communication.
- `-p 6060:6060`: Map port 6060 of the host to port 6060 of the container for pprof communication.
- `-p 8081:8081`: Map port 8081 of the host to port 8081 of the container for HTTP/1.1 REST communication.
- `--name ec-daemon-container`: Assign a name to the container for easier management.
- `ec-daemon`: The name of the Docker image to run.

## Verifying the Deployment

1. **Check Container Logs**

   To check the logs of the running container, use:

   ```bash
   docker logs ec-daemon-container
   ```

2. **Access the Exposed Ports**

   - **gRPC Communication**: Connect to `localhost:50050`.
   - **HTTP/1.1 REST Communication**: Access the REST API at `localhost:8081`.
   - **pprof Communication**: Access pprof at `localhost:6060`.

## Stopping the Container

To stop the running container, use:

```bash
docker stop ec-daemon-container
```

## Troubleshooting

If you encounter issues, consider the following:

- **Docker Daemon**: Ensure Docker is running correctly.
- **Container Logs**: Check logs using `docker logs` for errors or warnings.
- **Port Conflicts**: Ensure that the ports are not in use by other applications on your host.
