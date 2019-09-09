# Prometheus Service

This service has two responsibilites: 
1. configures Prometheus for monitoring services managed by keptn
1. takes alerts from Prometheus Alertmanager and translates the alert payload to a cloud event that is sent to the keptn eventbroker.
