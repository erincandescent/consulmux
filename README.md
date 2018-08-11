Simple Stupid Hacky Consul service proxy

This
 * Reads the first part of the hostname from the request
 * Looks it up as a local agent consul service
 * Proxies the request to the address specified

In combinatin with Consul's DNS proxy, this provides easy
access to services. For example, if you have Grafana configured
in Consul, then "http://grafana.service.consul" will open Grafana

# Why not Traefik or similar?
I should have probably used Traefik, but writing this was quicker,
and it does precisely the simple thing I wanted 
