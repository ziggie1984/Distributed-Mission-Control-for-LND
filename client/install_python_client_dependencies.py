import os
import subprocess
import sys
import shutil
from urllib.request import urlretrieve

# Directory where generated client code will be stored.
CLIENT_LNRPC = "lnrpc"

# Temporary directory for cloning Google APIs.
GOOGLE_APIS_DIR = os.path.join(os.getcwd(), "googleapis")

# Temporary files for lightning.proto and router.proto.
LIGHTNING_PROTO_FILE = os.path.join(os.getcwd(), "lightning.proto")
ROUTER_PROTO_FILE = os.path.join(os.getcwd(), "router.proto")

# Create the CLIENT_LNRPC directory if it doesn't exist.
if not os.path.exists(CLIENT_LNRPC):
    os.makedirs(CLIENT_LNRPC)

# Create a Python virtual environment named .ecrpc_client.
print("Creating Python virtual environment .ecrpc_client")
subprocess.run([sys.executable, "-m", "venv", ".ecrpc_client"])

# Activate the Python virtual environment.
print("Activating Python virtual environment")
if os.name == 'nt':
    activate_script = ".ecrpc_client\\Scripts\\activate"
else:
    activate_script = ".ecrpc_client/bin/activate"

# Define a helper function to run a command in the virtual environment.
def run_in_venv(command):
    activate_script = ""
    if os.name == 'nt':
        activate_script = ".ecrpc_client\\Scripts\\activate"
    else:
        activate_script = ".ecrpc_client/bin/activate"

    # Check if it is Windows OS.
    if os.name == 'nt':
        command = f"{activate_script} && {command}"
    else:
        # Otherwise it is Unix/Linux/Mac
        command = f"bash -c 'source {activate_script} && {command}'"

    subprocess.run(command, shell=True, check=True)

# Install the required Python packages.
print("Installing required Python packages")
run_in_venv(f"python -m pip install --upgrade pip")
run_in_venv(f"python -m pip install -r requirements.txt")

# Clone the Google APIs repository into the temporary directory.
print("Cloning Google APIs repository")
if os.path.exists(GOOGLE_APIS_DIR):
    shutil.rmtree(GOOGLE_APIS_DIR)
subprocess.run(["git", "clone", "https://github.com/googleapis/googleapis.git", GOOGLE_APIS_DIR])

# Download the lightning.proto file into the temporary directory.
print("Downloading lightning.proto")
urlretrieve("https://raw.githubusercontent.com/lightningnetwork/lnd/master/lnrpc/lightning.proto", LIGHTNING_PROTO_FILE)

# Generate the gRPC client code for lightning.proto.
print("Generating gRPC client code for lightning.proto")
subprocess.run([
    ".ecrpc_client/bin/python", "-m", "grpc_tools.protoc",
    f"--proto_path={GOOGLE_APIS_DIR}", f"--proto_path={os.getcwd()}",
    f"--python_out={CLIENT_LNRPC}", f"--grpc_python_out={CLIENT_LNRPC}",
    LIGHTNING_PROTO_FILE
])

# Download the router.proto file into the temporary directory.
print("Downloading router.proto")
urlretrieve("https://raw.githubusercontent.com/lightningnetwork/lnd/master/lnrpc/routerrpc/router.proto", ROUTER_PROTO_FILE)

# Generate the gRPC client code for router.proto.
print("Generating gRPC client code for router.proto")
subprocess.run([
    ".ecrpc_client/bin/python", "-m", "grpc_tools.protoc",
    f"--proto_path={GOOGLE_APIS_DIR}", f"--proto_path={os.getcwd()}",
    f"--python_out={CLIENT_LNRPC}", f"--grpc_python_out={CLIENT_LNRPC}",
    ROUTER_PROTO_FILE
])

# Clean up temporary files and directories.
print("Cleaning up temporary files and directories")
os.remove(LIGHTNING_PROTO_FILE)
os.remove(ROUTER_PROTO_FILE)
shutil.rmtree(GOOGLE_APIS_DIR)

print("Done")
