"""
REST Client for Mission Control Management between EC and LND

This script is designed to manage and integrate mission control data between an LND node and an External Coordinator (EC) server using RESTful API. It provides functionalities for secure communication, data querying, and data registration.
"""

import json
from typing import Tuple
import requests
import codecs

def get_secure_session(cert: str) -> requests.Session:
    """
    Creates a secure requests session using SSL credentials.

    Args:
        cert (str): Path to the SSL certificate file.

    Returns:
        requests.Session: A secure requests session.
    """
    session = requests.Session()
    session.verify = cert
    return session

def query_aggregated_mission_control(session: requests.Session, ec_rest_host: str) -> list:
    """
    Queries the aggregated mission control data from the External Coordinator server.

    Args:
        session (requests.Session): The secure requests session.
        ec_rest_host (str): The REST host address of the External Coordinator.

    Returns:
        list: A list of pairs from the aggregated mission control data.
    """
    url = f"https://{ec_rest_host}/v1/query_aggregated_mission_control"
    response = session.get(url, stream=True)
    response.raise_for_status()
    
    pairs = []
    for line in response.iter_lines():
        if line:
            data = json.loads(line.decode('utf-8'))
            pairs.extend(data["result"]["pairs"])
    return pairs

def register_mission_control(session: requests.Session, ec_rest_host: str, pairs: list) -> dict:
    """
    Registers mission control data with the External Coordinator.

    Args:
        session (requests.Session): The secure requests session.
        ec_rest_host (str): The REST host address of the External Coordinator.
        pairs (list): A list of pairs to register.

    Returns:
        dict: The response from the registration request.
    """
    url = f"https://{ec_rest_host}/v1/register_mission_control"
    data = {'pairs': pairs}
    response = session.post(url, json=data)
    response.raise_for_status()
    return response.json()

def query_mission_control_data_from_lnd(macaroon_path: str, tls_cert: str, lnd_rest_host: str) -> list:
    """
    Queries mission control data from the LND node.

    Args:
        macaroon_path (str): Path to the LND macaroon file.
        tls_cert (str): Path to the LND TLS certificate file.
        lnd_rest_host (str): The REST host address of the LND node.

    Returns:
        list: A list of mission control pairs from the LND node.
    """
    macaroon = codecs.encode(open(macaroon_path, 'rb').read(), 'hex')
    headers = {'Grpc-Metadata-macaroon': macaroon}
    url = f"https://{lnd_rest_host}/v2/router/mc"
    response = requests.get(url, headers=headers, verify=tls_cert)
    response.raise_for_status()
    return response.json().get('pairs', [])

def import_mission_control_data_into_lnd(macaroon_path: str, tls_cert: str, lnd_rest_host: str, pairs: list) -> bool:
    """
    Imports mission control data into the LND node.

    Args:
        macaroon_path (str): Path to the LND macaroon file.
        tls_cert (str): Path to the LND TLS certificate file.
        lnd_rest_host (str): The REST host address of the LND node.
        pairs (list): A list of pairs to import.

    Returns:
        bool: True if the import was successful, otherwise False.
    """
    macaroon = codecs.encode(open(macaroon_path, 'rb').read(), 'hex')
    headers = {'Grpc-Metadata-macaroon': macaroon}
    url = f"https://{lnd_rest_host}/v2/router/x/importhistory"
    data = {'pairs': pairs, 'force': False}
    response = requests.post(url, json=data, headers=headers, verify=tls_cert)
    response.raise_for_status()
    return response.status_code == 200

def register_my_lnd_mission_control_data_with_ec(lnd_macaroon_path: str, lnd_tls_cert: str, lnd_rest_host: str, ec_session: requests.Session, ec_rest_host: str) -> list:
    """
    Registers mission control data from the LND node with the External Coordinator.

    Args:
        lnd_macaroon_path (str): Path to the LND macaroon file.
        lnd_tls_cert (str): Path to the LND TLS certificate file.
        lnd_rest_host (str): The REST host address of the LND node.
        ec_session (requests.Session): The secure requests session for the External Coordinator.
        ec_rest_host (str): The REST host address of the External Coordinator.

    Returns:
        list: A list of mission control pairs registered into the External Coordinator.
    """
    mc_pairs = query_mission_control_data_from_lnd(
        lnd_macaroon_path, lnd_tls_cert, lnd_rest_host
    )
    register_mission_control(ec_session, ec_rest_host, mc_pairs)
    return mc_pairs

def import_mission_control_data_from_ec_to_my_lnd(ec_session: requests.Session,
ec_rest_host: str, lnd_macaroon_path: str, lnd_tls_cert: str,
lnd_rest_host: str) -> Tuple[bool, list]:
    """
    Imports mission control data from the External Coordinator to the LND node.

    Args:
        ec_session (requests.Session): The secure requests session for the External Coordinator.
        ec_rest_host (str): The REST host address of the External Coordinator.
        lnd_macaroon_path (str): Path to the LND macaroon file.
        lnd_tls_cert (str): Path to the LND TLS certificate file.
        lnd_rest_host (str): The REST host address of the LND node.

    Returns:
        Tuple[bool, list]: A tuple containing a boolean indicating success, and a list of mission control pairs imported into the LND node.
    """
    ec_pairs = query_aggregated_mission_control(ec_session, ec_rest_host)
    import_success = import_mission_control_data_into_lnd(
        lnd_macaroon_path, lnd_tls_cert, lnd_rest_host, ec_pairs,
    )
    return import_success, ec_pairs

if __name__ == "__main__":
    # Define configuration variables for the LND node.
    LND_REST_HOST = 'localhost:8080'
    LND_MACAROON_PATH = 'LND_DIR/data/chain/bitcoin/regtest/admin.macaroon'
    LND_TLS_CERT = 'LND_DIR/tls.cert'

    # Define configuration variables for the External Coordinator.
    EC_REST_HOST = 'localhost:8081'
    EC_TLS_CERT = "EC_DIR/tls.cert"

    # Create a secure session to communicate with the External Coordinator.
    ec_session = get_secure_session(cert=EC_TLS_CERT)

    # Register mission control data from the LND node with the External
    # Coordinator (EC).
    mc_pairs_registered = register_my_lnd_mission_control_data_with_ec(
        lnd_macaroon_path=LND_MACAROON_PATH, lnd_tls_cert=LND_TLS_CERT,
        lnd_rest_host=LND_REST_HOST, ec_session=ec_session,
        ec_rest_host=EC_REST_HOST
    )
    print((
        f"{len(mc_pairs_registered)} of your LND Mission Control pairs "
        "registered into EC 🎉"
    ))

    # Import mission control data from the External Coordinator (EC) to the LND
    # node.
    success, ec_pairs_imported = import_mission_control_data_from_ec_to_my_lnd(
        ec_session=ec_session, ec_rest_host=EC_REST_HOST,
        lnd_macaroon_path=LND_MACAROON_PATH, lnd_tls_cert=LND_TLS_CERT,
        lnd_rest_host=LND_REST_HOST
    )
    if success:
        print((
            f"{len(ec_pairs_imported)} EC Mission Control pairs "
            "imported into your LND 🎉"
        ))
