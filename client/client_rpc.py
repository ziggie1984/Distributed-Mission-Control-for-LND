"""
RPC Client for Mission Control Management between EC and LND

This script is designed to manage and integrate mission control data between an LND node and an External Coordinator (EC) server. It provides functionalities for secure gRPC communication, data querying, data registration, and integration with LND.
"""

import codecs
import os
import grpc
import ecrpc.external_coordinator_pb2 as ecrpc
import ecrpc.external_coordinator_pb2_grpc as ecrpcstub
import lnrpc.router_pb2 as routerrpc, lnrpc.router_pb2_grpc as routerstub

def get_secure_channel(target: str, cert: str) -> grpc.Channel:
    """
    Creates a secure gRPC channel using SSL credentials.

    Args:
        target (str): The target server address.
        cert (str): Path to the SSL certificate file.

    Returns:
        grpc.Channel: A secure gRPC channel.
    """
    with open(cert, 'rb') as f:
        trusted_certs = f.read()
    credentials = grpc.ssl_channel_credentials(root_certificates=trusted_certs)
    return grpc.secure_channel(target, credentials)

def query_aggregated_mission_control(stub) -> list:
    """
    Queries the aggregated mission control data from the External Coordinator server using server-side streaming.

    Args:
        stub: The gRPC stub for the External Coordinator.

    Returns:
        list: A list of pairs from the aggregated mission control data.
    """
    request = ecrpc.QueryAggregatedMissionControlRequest()
    pairs = []
    try:
        for response in stub.QueryAggregatedMissionControl(request):
            pairs.extend(response.pairs)
    except Exception as e:
        print(f"Failed to process streaming response: {e}")
    return pairs

def register_mission_control(stub, pairs: list[routerrpc.PairHistory]) -> ecrpc.RegisterMissionControlResponse:
    """
    Registers mission control data with the External Coordinator.

    Args:
        stub: The gRPC stub for the External Coordinator.
        pairs (list): A list of `routerrpc.PairHistory` objects to register.

    Returns:
        ecrpc.RegisterMissionControlResponse: The response from the registration request.
    """
    converted_pairs = [convert_to_ecrpc_pair_history(pair) for pair in pairs]
    request = ecrpc.RegisterMissionControlRequest(pairs=converted_pairs)
    response = stub.RegisterMissionControl(request)
    return response

def query_mission_control_data_from_lnd(stub) -> list[routerrpc.PairHistory]:
    """
    Queries mission control data from the LND node.

    Args:
        stub: The gRPC stub for the LND router.

    Returns:
        list[routerrpc.PairHistory]: A list of mission control pairs from the LND node.
    """
    request = routerrpc.QueryMissionControlRequest()
    response = stub.QueryMissionControl(request)
    return response.pairs

def import_mission_control_data_into_lnd(stub, pairs: list[ecrpc.PairHistory]) -> bool:
    """
    Imports mission control data into the LND node.

    Args:
        stub: The gRPC stub for the LND router.
        pairs (list): A list of `ecrpc.PairHistory` objects to import.

    Returns:
        bool: True if the import was successful, otherwise False.
    """
    converted_pairs = []
    for pair in pairs:
        converted_pairs.append(convert_to_routerrpc_pair_history(pair))

    request = routerrpc.XImportMissionControlRequest(
        pairs=converted_pairs,
        force=False
    )
    _ = stub.XImportMissionControl(request)
    return True

def get_lnd_router_stub(lnd_macaroon_path: str, lnd_tls_cert: str, lnd_grpc_host: str) -> routerstub.RouterStub:
    """
    Creates a gRPC stub for the LND router with secure credentials and macaroon authentication.

    Args:
        lnd_macaroon_path (str): Path to the LND macaroon file.
        lnd_tls_cert (str): Path to the LND TLS certificate file.
        lnd_grpc_host (str): The gRPC host address of the LND node.

    Returns:
        routerstub.RouterStub: A gRPC stub for the LND router.
    """
    # Read and encode the macaroon file as hex
    macaroon = codecs.encode(open(lnd_macaroon_path, 'rb').read(), 'hex')
    
    def metadata_callback(context, callback):
        """
        Callback function to add the macaroon to the gRPC metadata.

        Args:
            context: The gRPC context.
            callback: The callback function to invoke with the metadata.
        """
        callback([('macaroon', macaroon)], None)
    
    # Create gRPC metadata credentials using the macaroon
    auth_creds = grpc.metadata_call_credentials(metadata_callback)
    
    # Set the gRPC SSL cipher suites environment variable
    os.environ['GRPC_SSL_CIPHER_SUITES'] = 'HIGH+ECDSA'
    
    # Read the TLS certificate file
    cert = open(lnd_tls_cert, 'rb').read()
    
    # Create SSL credentials using the TLS certificate
    ssl_creds = grpc.ssl_channel_credentials(cert)

    # Combine the SSL and metadata (macaroon) credentials
    combined_creds = grpc.composite_channel_credentials(ssl_creds, auth_creds)

    # Create a secure gRPC channel with the combined credentials
    channel = grpc.secure_channel(lnd_grpc_host, combined_creds)

    # Create and return a gRPC stub for the LND router
    stub = routerstub.RouterStub(channel)
    return stub

def register_my_lnd_mission_control_data_with_ec(lnd_router_stub, ec_stub) -> list[routerrpc.PairHistory]:
    """
    Registers mission control data from the LND node with the External Coordinator.

    Args:
        lnd_router_stub: The gRPC stub for the LND router.
        ec_stub: The gRPC stub for the External Coordinator.

    Returns:
        list: A list of mission control pairs registered into the External Coordinator.
    """
    mc_pairs = query_mission_control_data_from_lnd(lnd_router_stub)
    register_mission_control(ec_stub, mc_pairs)
    return mc_pairs

def import_mission_control_data_from_ec_to_my_lnd(lnd_router_stub, ec_stub) -> list:
    """
    Imports mission control data from the External Coordinator to the LND node.

    Args:
        lnd_router_stub: The gRPC stub for the LND router.
        ec_stub: The gRPC stub for the External Coordinator.

    Returns:
        list: A list of mission control control pairs imported into the LND node.
    """
    ec_pairs = query_aggregated_mission_control(stub=ec_stub)
    import_mission_control_data_into_lnd(lnd_router_stub, ec_pairs)
    return ec_pairs

def convert_to_ecrpc_pair_history(pair: routerrpc.PairHistory) -> ecrpc.PairHistory:
    """
    Converts a `routerrpc.PairHistory` object to an `ecrpc.PairHistory` object.

    Args:
        pair (routerrpc.PairHistory): The pair history object to convert.

    Returns:
        ecrpc.PairHistory: The converted pair history object.
    """
    return ecrpc.PairHistory(
        node_from=pair.node_from,
        node_to=pair.node_to,
        history=ecrpc.PairData(
            fail_time=pair.history.fail_time,
            fail_amt_sat=pair.history.fail_amt_sat,
            fail_amt_msat=pair.history.fail_amt_msat,
            success_time=pair.history.success_time,
            success_amt_sat=pair.history.success_amt_sat,
            success_amt_msat=pair.history.success_amt_msat,
        )
    )

def convert_to_routerrpc_pair_history(pair: ecrpc.PairHistory) -> routerrpc.PairHistory:
    """
    Converts an `ecrpc.PairHistory` object to a `routerrpc.PairHistory` object.

    Args:
        pair (ecrpc.PairHistory): The pair history object to convert.

    Returns:
        routerrpc.PairHistory: The converted pair history object.
    """
    return routerrpc.PairHistory(
        node_from=pair.node_from,
        node_to=pair.node_to,
        history=routerrpc.PairData(
            fail_time=pair.history.fail_time,
            fail_amt_sat=pair.history.fail_amt_sat,
            fail_amt_msat=pair.history.fail_amt_msat,
            success_time=pair.history.success_time,
            success_amt_sat=pair.history.success_amt_sat,
            success_amt_msat=pair.history.success_amt_msat,
        )
    )

if __name__ == "__main__":
    # Define configuration variables for the LND node.
    LND_GRPC_HOST = 'localhost:10009'
    LND_MACAROON_PATH = 'LND_DIR/data/chain/bitcoin/regtest/admin.macaroon'
    LND_TLS_CERT = 'LND_DIR/tls.cert'

    # Create a stub to communicate with the LND node.
    lnd_router_stub = get_lnd_router_stub(
        LND_MACAROON_PATH, LND_TLS_CERT, LND_GRPC_HOST,
    )

    # Define configuration variables for the External Coordinator.
    EC_TLS_CERT = "ExternalCoordinator_DIR/tls.cert"
    EC_GRPC_HOST = "localhost:50050"

    # Create a secure channel and stub to communicate with the External
    # Coordinator.
    ec_channel = get_secure_channel(
        target=EC_GRPC_HOST, cert=EC_TLS_CERT,
    )
    ec_stub = ecrpcstub.ExternalCoordinatorStub(ec_channel)

    # Register mission control data from the LND node with the External
    # Coordinator (EC).
    mc_pairs_registered = register_my_lnd_mission_control_data_with_ec(
        lnd_router_stub, ec_stub,
    )
    print((
        f"{len(mc_pairs_registered)} of your LND Mission Control pairs "
        "registered into EC 🎉"
    ))

    # Import mission control data from the External Coordinator (EC) to the LND
    # node.
    ec_pairs_imported = import_mission_control_data_from_ec_to_my_lnd(
        lnd_router_stub, ec_stub,
    )
    print((
        f"{len(ec_pairs_imported)} EC Mission Control pairs "
        "imported into your LND 🎉"
    ))
