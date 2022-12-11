# Hive Taiko RPC test suite

This test suite is a copy of the ETH L1 RPC test suite adapted for Taiko L2.
It tests several real-world scenarios such as sending value transactions,
deploying a contract or interacting with one.

./hive --sim=taiko/rpc --client=taiko-l1,taiko-geth,taiko-protocol,taiko-client --docker.output
