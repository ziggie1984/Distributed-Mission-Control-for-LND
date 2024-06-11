# Project Impact

## Overview
The "Distributed-Mission-Control-for-LND" project aims to address critical
challenges faced by non-custodial wallets like Blixt and Breez. This project
focuses on developing a dynamic system that enables LND node operators to
register their Mission Control (MC) data with an External Coordinator (EC)
daemon service. By aggregating and tailoring this data, the EC service allows
clients, such as non-custodial wallets, to query and utilize it effectively as
part of their internal mission control state.

The primary goal is to enhance the efficiency and reliability of
the Lightning Network by:

- **Drastically reducing the rate of failed attempts**: Access to comprehensive
route information helps in avoiding paths likely to fail, thus minimizing failed
payment attempts.
- **Increasing the payment success rate**: By using aggregated MC data, wallets
can select more reliable routes, leading to a higher rate of successful payments.
- **Reducing payment processing time**: Efficient route selection based on MC
data ensures quicker payment processing, enhancing user experience.
- **Resolving stuck HTLCs**: Providing updated and accurate route information
helps in mitigating issues with Hash Time-Locked Contracts (HTLCs) getting
stuck, ensuring smoother transactions.
- **Reducing force close operations**: By enhancing route reliability and
payment success rates, the need for force-closing channels due to failed
transactions or stuck HTLCs is significantly reduced.
- **Reducing Bootstrapping Time**: By querying recent data from the external
coordination service, clients can operate more efficiently from the start,
minimizing initial failed payments and improving route estimation accuracy.

## Problem Statement
Non-custodial wallets in the Lightning Network often face significant issues
during their initial setup, known as the cold start problem. This problem is
characterized by a high number of failed payment attempts and inefficient route
selections due to the lack of historical data. The
"Distributed-Mission-Control-for-LND" project aims to solve this by creating
a system where LND node operators can register their MC data with an EC daemon
service. This service provides a mechanism for clients to access recent and
relevant MC data, thereby improving the overall efficiency and reliability of
the Lightning Network for non-custodial wallets.

By addressing these challenges, the project aims to significantly improve the
operational efficiency and reliability of non-custodial wallets within
the Lightning Network.

## Impact
The "Distributed-Mission-Control-for-LND" project is expected to have several
positive impacts:

1. **Enhanced Reliability and Efficiency**: By providing access to aggregated
and tailored MC data, the project will help non-custodial wallets improve their
route selection process, leading to fewer failed attempts and faster payment
processing.
2. **Improved User Experience**: Reduced payment processing time and higher
success rates will enhance the overall user experience for non-custodial wallet
users.
3. **Increased Privacy**: Unlike centralized pathfinding services, this project
will allow for hot injecting of critical routing data while preserving sender
privacy, offering a significant advantage over existing solutions.
4. **Scalability to Routing Nodes**: The service can potentially be extended to
routing nodes, enabling them to either sell routing data or use the service to
improve their routing probability, with proper data validation mechanisms to
prevent fraudulent data.

## Metrics
The impact of the project will be measured using the following metrics:

1. **Reduction in Failed Payment Attempts**: Tracking the number of failed
payment attempts before and after implementing the service.
2. **Payment Success Rate**: Measuring the percentage increase in successful
payments.
3. **Average Payment Processing Time**: Monitoring the reduction in time taken
to process payments.
4. **User Satisfaction**: Conducting surveys and collecting feedback from
non-custodial wallet users regarding their experience.
5. **Adoption Rate**: Number of non-custodial wallets and routing nodes using
the EC service.
6. **Privacy Metrics**: Evaluating the level of sender privacy maintained
compared to centralized pathfinding services.

## Long-term Vision
The long-term vision for the "Distributed-Mission-Control-for-LND" project is
to create a sustainable and scalable solution that significantly improves the
efficiency, reliability, and privacy of the Lightning Network. Initially
targeting non-custodial wallets, the service aims to solve the cold start
problem and enhance route selection processes. In the future, the service can be
extended to routing nodes, allowing them to sell or acquire routing data to
further improve the network's reliability. Ensuring robust data validation
mechanisms will be crucial to maintaining the integrity and trustworthiness of
the service.

## Comparison with Centralized Pathfinding Services

### Centralized Pathfinding Services
Centralized pathfinding services typically operate as follows:

- **Centralized Control**: A single entity manages the pathfinding process,
collecting data from multiple nodes to determine the best routes for
transactions.
- **Data Centralization**: All route information and transaction data are sent
to a central server, where the best route is computed.
- **Privacy Concerns**: Centralized services can log detailed information about
each transaction, including sender and recipient details, amounts, and routing
paths. This centralization poses significant privacy risks as the data could be
exposed, misused, or subject to surveillance.
- **Efficiency**: These services can be highly efficient due to their access to
a vast amount of network data, but this comes at the cost of privacy and
potential single points of failure.

### Distributed-Mission-Control-for-LND Approach
In contrast, the "Distributed-Mission-Control-for-LND" project operates on
a decentralized model:

- **Decentralized Control**: Instead of relying on a single entity, the project
enables multiple LND node operators to register their Mission Control (MC) data
with an External Coordinator (EC) daemon service.
- **Data Distribution**: Route information is aggregated from various sources,
reducing reliance on a central server. Each EC service can independently provide
pathfinding assistance.
- **Enhanced Privacy**: By querying aggregated MC data from decentralized
services rather than sending detailed transaction data to a central server,
the sender's privacy is significantly enhanced. This approach minimizes the risk
of data exposure and misuse.
- **Efficiency with Privacy**: While maintaining a high level of efficiency in
route selection, the distributed approach ensures that sender information
remains confidential, balancing efficiency with robust privacy protections.

### Key Differences
Here are the key differences between Centralized Services and Distributed Mission Control for LND Approach;
| **Aspect**                  | **Centralized Services**                                       | **Distributed Approach**                                                                       |
|-----------------------------|----------------------------------------------------------------|------------------------------------------------------------------------------------------------|
| **Control and Management**  | Single entity control                                          | Multiple, independent node operators                                                           |
| **Data Handling**           | Centralized data collection and processing                     | Decentralized data aggregation and querying                                                    |
| **Privacy**                 | Higher privacy risks due to data centralization                | Enhanced privacy by reducing the need to expose detailed transaction data to any single entity |
| **Single Point of Failure** | Vulnerable to failures or attacks on the central server        | More resilient, as multiple EC services can operate independently                              |
| **Scalability**             | May face scalability issues due to the load on a single server | Naturally scalable as more node operators can join and contribute data                         |

## Conclusion
The "Distributed-Mission-Control-for-LND" project addresses critical challenges
faced by non-custodial wallets in the Lightning Network. By providing a dynamic
system for registering and querying MC data, the project aims to drastically
reduce failed payment attempts, increase success rates, and improve overall user
experience. The long-term vision includes scaling the service to routing nodes
and ensuring sustainable impact through robust data validation mechanisms. This
project holds the potential to significantly enhance the efficiency,
reliability, and privacy of the Lightning Network, benefiting both wallet users
and node operators.