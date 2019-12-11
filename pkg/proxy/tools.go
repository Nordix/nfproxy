package proxy

import (
	"crypto/sha256"
	"encoding/base32"
)

func portProtoHash(servicePortName string, protocol string) string {
	hash := sha256.Sum256([]byte(servicePortName + protocol))
	encoded := base32.StdEncoding.EncodeToString(hash[:])
	return "k8s-nfproxy-svc-" + encoded[:16]
}

// servicePortChainName takes the ServicePortName for a service and
// returns the associated iptables chain.  This is computed by hashing (sha256)
// then encoding to base32 and truncating with the prefix "KUBE-SVC-".
//func servicePortChainName(servicePortName string, protocol string) utiliptables.Chain {
//	return utiliptables.Chain("KUBE-SVC-" + portProtoHash(servicePortName, protocol))
//}

// serviceFirewallChainName takes the ServicePortName for a service and
// returns the associated iptables chain.  This is computed by hashing (sha256)
// then encoding to base32 and truncating with the prefix "KUBE-FW-".
//func serviceFirewallChainName(servicePortName string, protocol string) utiliptables.Chain {
//	return utiliptables.Chain("KUBE-FW-" + portProtoHash(servicePortName, protocol))
//}

// serviceLBPortChainName takes the ServicePortName for a service and
// returns the associated iptables chain.  This is computed by hashing (sha256)
// then encoding to base32 and truncating with the prefix "KUBE-XLB-".  We do
// this because IPTables Chain Names must be <= 28 chars long, and the longer
// they are the harder they are to read.
//func serviceLBChainName(servicePortName string, protocol string) utiliptables.Chain {
//	return utiliptables.Chain("KUBE-XLB-" + portProtoHash(servicePortName, protocol))
//}

// This is the same as servicePortChainName but with the endpoint included.
func servicePortEndpointChainName(servicePortName string, protocol string, endpoint string) string {
	hash := sha256.Sum256([]byte(servicePortName + protocol + endpoint))
	encoded := base32.StdEncoding.EncodeToString(hash[:])
	return "k8s-nfproxy-sep-" + encoded[:16]
}
