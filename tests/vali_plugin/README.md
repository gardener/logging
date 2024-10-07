# Plugin test

This test simulates creating 100 clusters with a logger pod in each cluster, 
verifying that the produced log volume is fully accounted. 
The test creates an instance of a fluent-bit output plugin with a k8s informer 
processing the cluster lifecycle events. Each cluster resource creates a dedicated output
client responsible for packing and sending the logs to the simulated backend.
Finally, the test counts the received logs in the backend and checks the total volume.

The test verifies the following plugin components:
- the plugin controller maintaining a list of clients corresponding to the cluster resources
- the seed and shoots client decorator chains
- the correct packaging of logs into respective backend streams
