# Distributed-Mission-Control-for-LND

## Installation Instructions

### Step 1: Install Go 1.22.2
1. Visit the official Go download page: [Go Downloads](https://go.dev/dl).
2. Download and install the appropriate version for your OS and hardware architecture.

### Step 2: Add GOBIN Path to Your $PATH
1. Open your terminal.
2. Add the following line to your shell profile file (e.g., `~/.bashrc`, `~/.zshrc`, `~/.profile`):

```sh
export PATH=$PATH:$(go env GOBIN)
```
3. Reload your shell profile:

```sh
source ~/.bashrc
```

### Step 3: Install Buf
1. Run the `install_buf.sh` script with `sudo` to ensure it has the necessary permissions to install the binary in `/usr/local/bin`:

```sh
sudo ./scripts/install_buf.sh
```

### Step 4: Install Protobuf Plugins
1. Run the `install_protobuf_plugins.sh` script:

```sh
./scripts/install_protobuf_plugins.sh
```

### Step 5: Install Make Command
1. Ensure `make` is installed on your system. On most Unix-based systems, `make` is pre-installed. If not, install it using your package manager.
    - **Ubuntu/Debian**: `sudo apt-get install build-essential`
    - **MacOS**: `xcode-select --install`

### Step 6: Build the EC Daemon (ec-debug)
1. Run the following command to build the EC Daemon:

```sh
make build
```

### Step 7: Install the EC Daemon (ec)
1. Run the following command to install the EC Daemon in the Go bin directory, allowing you to run it using the command `ec`:

```sh
make install
```

### Step 8: Run the Test Cases
1. Run the following command to execute the test cases:

```sh
make test
```
