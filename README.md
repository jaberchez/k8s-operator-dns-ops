# Overview

This is an example of Kubernetes Operator created with the [Operator SDK](https://github.com/operator-framework/operator-sdk)

It watches the Kubernetes Node Resource and when a Node is created, updated or deleted, it creates, updates or deletes the DNS A record

In this example [PowerDNS](https://doc.powerdns.com/) is used because it provides us with a web user interface and a REST API

## Testing
To test the operator an OpenShift 4.x environment is preferred because you can scale Nodes via MachineSet
```bash
oc scale machineset machineset-name -n openshift-machine-api --replicas 2
```

## Download and Install Operator SDK

```bash
wget https://github.com/operator-framework/operator-sdk/releases/download/v1.13.0/operator-sdk_linux_amd64
chmod +x operator-sdk_linux_amd64
sudo mv operator-sdk_linux_amd64 /usr/local/bin/operator-sdk
```

## Operator

```bash
operator-sdk init --domain example.com --repo github.com/jaberchez/k8s-operator-dns-ops
operator-sdk create api --group=core --version=v1 --kind=Node --controller=true --resource=false
```

**Note:** Since a Node Resource is used (not a CRD) the _--resource_ parameter must be _false_

## Deploy PowerDNS
```bash
oc new-project powerdns
oc apply -f deploy/powerdns

oc adm policy add-scc-to-user anyuid -z powerdns
oc adm policy add-scc-to-user anyuid -z powerdns-webui
```

**Note:** PowerDNS needs a MySQL server. For this example a ephemeral storage is used

## Build operator docker image
```bash
cd operator
sudo podman build -t quay.io/jberchez-redhat/k8s-operator-dns-ops:v1.0 .
```

## Push the docker image into the registry
```bash
sudo podman push quay.io/jberchez-redhat/k8s-operator-dns-ops:v1.0
```

## Deploy Operator
```bash
oc new-project k8s-operator-dns-ops
oc apply -f deploy/operator
```

## TODO
Add more DNS servers like [Infoblox](https://www.infoblox.com/)

## Sources
[Operator SDK](https://github.com/operator-framework/operator-sdk)

[Kubebuilder Book](https://book.kubebuilder.io/quick-start.html)

[PowerDNS](https://doc.powerdns.com/)


