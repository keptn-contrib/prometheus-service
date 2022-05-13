# Troubleshooting

This file contains common errors and their resolutions.

## Not enough permissions / is forbidden

Permission errors like the following can happen when you forget to apply the `prometheus-service` role binding to your cluster.

```
$ keptn configure monitoring prometheus --project=sockshop --service=carts
ID of Keptn context: f02e0fc3-e148-497c-a312-31aa712d9feb
configmaps "prometheus-server" is forbidden: User "system:serviceaccount:keptn:keptn-prometheus-service" cannot get resource "configmaps" in API group "" in the namespace "monitoring"
configmaps "prometheus-server" is forbidden: User "system:serviceaccount:keptn:keptn-prometheus-service" cannot get resource "configmaps" in API group "" in the namespace "monitoring"
configmaps "prometheus-server" is forbidden: User "system:serviceaccount:keptn:keptn-prometheus-service" cannot get resource "configmaps" in API group "" in the namespace "monitoring"

# Or

$ keptn configure monitoring prometheus --project=sockshop --service=carts
ID of Keptn context: 357de774-32ec-4e03-99e1-5ae726185962
not enough permissions to access configmap. Check if the role binding is correct
```

To fix this you need to apply the role binding in your cluster:

```
kubectl apply -f https://raw.githubusercontent.com/keptn-contrib/prometheus-service/<VERSION>/deploy/role.yaml -n monitoring
```