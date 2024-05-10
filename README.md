# Distributed-Mission-Control for LND
This project creates an external coordination service which aggregates Mission Control Data from different lightning nodes and makes the data available via an API.

Mission Control is LND's central path finding brain. Overtime, clients will populate this brain with empirical observations, in order to derive a more accurate model of the path finding network at a given instance. Today all clients need to deal with a cold start issue where to start with, they have no observations. The “XImportMissionControl” API call in LND can be used to allow clients to start with a hot cache.
This project would use LND’s Mission Control API calls to implement a dynamic system where clients export their mission control observations to a coordination service that then figures out how to intelligently merge the observations into a unified data set. This can then be used to implement fast bootstrap for LND nodes and increase payment success rates.


This project is part of the https://github.com/lightningnetwork ecosystem.
