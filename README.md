# HFP
HEP Fidelity Proxy

Reliable way of relaying all your HEP to any HEP remote server that is behind unreliable networks.

It is buffered TCP proxy with option of storing HEP locally in cases of backend HEP server unavailability and replaying of that HEP after HEP server becomes reachable again. It can be beneficial in highly distributed voice networks to reliably deliver your HEP to its destination.

Usage: ./HFP -l :9060 -r (HEP server we want to reliably proxy HEP)

