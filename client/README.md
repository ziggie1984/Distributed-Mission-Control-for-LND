# How to Use the Mission Control Management Clients

This document provides instructions on how to use the REST and RPC clients for managing mission control data between an LND node and an External Coordinator (EC) server.

## Table of Contents
- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Usage](#usage)
  - [Configuration Variables](#configuration-variables)
  - [Setting Up Secure Sessions](#setting-up-secure-sessions)
  - [Querying Aggregated Mission Control Data](#querying-aggregated-mission-control-data)
  - [Registering Mission Control Data](#registering-mission-control-data)
  - [Querying Mission Control Data from LND](#querying-mission-control-data-from-lnd)
  - [Importing Mission Control Data into LND](#importing-mission-control-data-into-lnd)
  - [Registering LND Mission Control Data with EC](#registering-lnd-mission-control-data-with-ec)
  - [Importing Mission Control Data from EC to LND](#importing-mission-control-data-from-ec-to-lnd)

## Overview

The provided scripts, `client_rpc.py` and `client_rest.py`, are designed to
manage and integrate mission control data between an LND node and an External
Coordinator (EC) server using RESTful APIs.

## Prerequisites

Before using the clients, ensure you have the following:
- Python installed on your system
(ensure compatibility with dependencies listed in `requirements.txt`)
- SSL certificates for secure communication
- Access to LND node's macaroon file and TLS certificate
- Access to EC node's TLS certificate

## Installation

1. **Run the installation script**:

   The `install_python_client_dependencies.py` script will automatically create
   a virtual environment, install the required dependencies, and generate the
   necessary gRPC client code.

   ```bash
   python install_python_client_dependencies.py
   ```

2. **Activate the Python virtual environment**:

   ```bash
   source .ecrpc_client/bin/activate
   ```

## Usage

### Configuration Variables

Before running the clients, update the configuration variables in your script to
match your environment:

- **LND Node Configuration**:
  - `LND_RPC_PORT`: The RPC port of the LND node (e.g., `10009`)
  - `LND_REST_HOST`: The REST host address of the LND node
  (e.g., `localhost:8080`)
  - `LND_MACAROON_PATH`: Path to the LND macaroon file
  s(e.g., `LND_DIR/data/chain/bitcoin/regtest/admin.macaroon`)
  - `LND_TLS_CERT`: Path to the LND TLS certificate file
  (e.g., `LND_DIR/tls.cert`)

- **External Coordinator Configuration**:
  - `EC_RPC_PORT`: The RPC port of the External Coordinator (e.g., `50050`)
  - `EC_REST_HOST`: The REST host address of the External Coordinator
  (e.g., `localhost:8081`)
  - `EC_TLS_CERT`: Path to the SSL certificate file for the External Coordinator
  (e.g., `ExternalCoordinator_DIR/tls.cert`)

### Setting Up Secure Sessions

Create a secure requests session using SSL credentials.

### Querying Aggregated Mission Control Data

Query aggregated mission control data from the EC server.

### Registering Mission Control Data

Register mission control data with the EC server.

### Querying Mission Control Data from LND

Query mission control data from the LND node.

### Importing Mission Control Data into LND

Import mission control data into the LND node.

### Registering LND Mission Control Data with EC

Register mission control data from the LND node with the EC.

### Importing Mission Control Data from EC to LND

Import mission control data from the EC server to the LND node.