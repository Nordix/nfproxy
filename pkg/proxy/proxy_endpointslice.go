/*
Copyright 2020 The nfproxy Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package proxy

import (
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	discovery "k8s.io/api/discovery/v1beta1"
	"k8s.io/klog"
)

func getServiceNameFromServiceNameLabel(labels map[string]string) (string, bool) {
	name, ok := labels[discovery.LabelServiceName]
	if !ok {
		return "", false
	}

	return name, true
}

func processEpSlice(epsl *discovery.EndpointSlice) ([]epInfo, error) {
	var ports []epInfo
	svcName, found := getServiceNameFromServiceNameLabel(epsl.ObjectMeta.Labels)
	if !found {
		// Slice does not have "kubernetes.io/service-name" label
		return ports, nil
	}
	for _, e := range epsl.Endpoints {
		var svcPortName ServicePortName
		for _, p := range epsl.Ports {
			if *p.Port == 0 {
				return nil, fmt.Errorf("found invalid endpoint slice port %s", *p.Name)
			}
			svcPortName = getSvcPortName(svcName, epsl.Namespace, *p.Name, *p.Protocol)
			for _, addr := range e.Addresses {
				port := epInfo{
					name: svcPortName,
					addr: &v1.EndpointAddress{
						IP:        addr,
						TargetRef: e.TargetRef,
						NodeName:  e.Hostname,
					},
					port: &v1.EndpointPort{},
				}
				if e.Hostname != nil {
					port.addr.Hostname = *e.Hostname
				}
				if p.Name != nil {
					port.port.Name = *p.Name
				}
				if p.Port != nil {
					port.port.Port = *p.Port
				}
				if p.Protocol != nil {
					port.port.Protocol = *p.Protocol
				}
				if *e.Conditions.Ready {
					port.ready = true
				}
				ports = append(ports, port)
			}
		}
	}

	return ports, nil
}

func (p *proxy) AddEndpointSlice(epsl *discovery.EndpointSlice) {
	s := time.Now()
	defer klog.V(5).Infof("AddEndpointSlice for a EndpointSlice %s/%s ran for: %d nanoseconds", epsl.Namespace, epsl.Name, time.Since(s))
	p.cache.storeEpSlInCache(epsl)
	klog.V(5).Infof("AddEndpointSlice for a EndpointSlice %s/%s", epsl.Namespace, epsl.Name)
	klog.V(6).Infof("Endpoints: %+v Ports: %+v Address type: %+v", epsl.Endpoints, epsl.Ports, epsl.AddressType)

	info, err := processEpSlice(epsl)
	if err != nil {
		klog.Errorf("failed to process Endpoint slice %s/%s with error: %+v", epsl.Namespace, epsl.Name, err)
		return
	}

	for _, e := range info {
		// Skipping not ready port, will program chains/rules once it becomes ready.
		if !e.ready {
			klog.V(5).Infof("Skip not Ready port %+v in Endpoint Slice %s/%s", e.port, epsl.Namespace, epsl.Name)
			continue
		}
		klog.V(5).Infof("adding Endpoint Slice %s/%s port %+v", epsl.Namespace, epsl.Name, e.port)
		if err := p.addEndpoint(e.name, e.addr, e.port); err != nil {
			klog.Errorf("failed to add Endpoint Slice %s/%s port %+v with error: %+v", epsl.Namespace, epsl.Name, e.port, err)
			return
		}
	}
}

func (p *proxy) DeleteEndpointSlice(epsl *discovery.EndpointSlice) {
	s := time.Now()
	defer klog.V(5).Infof("DeleteEndpointSlice for a EndpointSlice %s/%s ran for: %d nanoseconds", epsl.Namespace, epsl.Name, time.Since(s))
	klog.V(5).Infof("DeleteEndpointSlice for a EndpointSlice %s/%s", epsl.Namespace, epsl.Name)
	klog.V(6).Infof("Endpoints: %+v Ports: %+v Address type: %+v", epsl.Endpoints, epsl.Ports, epsl.AddressType)
	info, err := processEpSlice(epsl)
	if err != nil {
		klog.Errorf("failed to process Endpoint slice %s/%s with error: %+v", epsl.Namespace, epsl.Name, err)
		return
	}
	for _, e := range info {
		// Skip ping not ready port, all related chains/rules were either never created, if port has never been ready
		// or during EndpointSlice update when port went from Ready to Not Ready.
		if !e.ready {
			klog.V(5).Infof("Skip not Ready port %+v in Endpoint Slice %s/%s", e.port, epsl.Namespace, epsl.Name)
			continue
		}
		p.mu.Lock()
		eps, ok := p.endpointsMap[e.name]
		p.mu.Unlock()
		if !ok {
			continue
		}
		klog.V(5).Infof("Removing Endpoint Slice %s/%s port %+v", epsl.Namespace, epsl.Name, e.port)
		if err := p.deleteEndpoint(e.name, e.addr, e.port, eps); err != nil {
			klog.Errorf("failed to remove Endpoint Slice %s/%s port %+v with error: %+v", epsl.Namespace, epsl.Name, e.port, err)
			continue
		}
	}
	p.cache.removeEpSlFromCache(epsl.Name, epsl.Namespace)
}

func (p *proxy) UpdateEndpointSlice(epslOld, epslNew *discovery.EndpointSlice) {
	s := time.Now()
	defer klog.V(5).Infof("UpdateEndpointSlice for a EndpointSlice %s/%s ran for: %d nanoseconds", epslNew.Namespace, epslNew.Name, time.Since(s))
	klog.V(5).Infof("UpdateEndpointSlice for a EndpointSlice %s/%s Address type: %+v", epslNew.Namespace, epslNew.Name, epslNew.AddressType)
	klog.V(6).Infof("Endpoints Old: %+v Endpoints New: %+v", epslOld.Endpoints, epslNew.Endpoints)
	klog.V(6).Infof("Ports Old: %+v Ports New: %+v", epslOld.Ports, epslNew.Ports)
	var storedEpSl *discovery.EndpointSlice
	ver, err := p.cache.getCachedEpSlVersion(epslNew.Name, epslNew.Namespace)
	if err != nil {
		klog.Errorf("UpdateEndpoint did not find Endpoint Slice %s/%s in cache, it is a bug, please file an issue", epslNew.Namespace, epslNew.Name)
		storedEpSl = epslOld
	} else {
		// TODO add logic to check version, if oldEp's version more recent than storedEp, then use oldEp as the most current old object.
		oldVer := epslOld.ObjectMeta.GetResourceVersion()
		if oldVer != ver {
			klog.Warningf("mismatch version detected between old Endpoint Slice %s/%s and last known stored in cache %s/%s",
				epslNew.Namespace, epslNew.Name, oldVer, ver)
		}
		storedEpSl, _ = p.cache.getLastKnownEpSlFromCache(epslNew.Name, epslNew.Namespace)
	}

	// Check for new Endpoint's ports, if found adding them into EndpointMap and corresponding programming rules.
	info, err := processEpSlice(epslNew)
	if err != nil {
		klog.Errorf("failed to update Endpoint Slice %s/%s with error: %+v", epslNew.Namespace, epslNew.Name, err)
		return
	}
	for _, e := range info {
		oldReady, found := isPortInEndpointSlice(storedEpSl, e.port, e.addr)
		if !found && e.ready {
			// Case when port and address are not in the cache and new endpoint is in Ready state, so add new port
			klog.V(5).Infof("adding Endpoint Slice %s/%s port: %+v", epslNew.Namespace, epslNew.Name, *e.port)
			if err := p.addEndpoint(e.name, e.addr, e.port); err != nil {
				klog.Errorf("failed to update Endpoint Slice %s/%s port %+v with error: %+v", epslNew.Namespace, epslNew.Name, *e.port, err)
			}
			continue
		}
		if !found && !e.ready {
			// Case when port and address are not in the cache and new endpoint is NOT in Ready state, do nothing
			continue
		}
		if found && e.ready && oldReady {
			// Case when nothing changed for port and address pair, ignoring it
			continue
		}
		if found && !e.ready && oldReady {
			// Case when existing Endpoint state got changed from Ready to NOT Ready
			p.mu.Lock()
			eps, ok := p.endpointsMap[e.name]
			p.mu.Unlock()
			if !ok {
				continue
			}
			klog.V(5).Infof("removing Endpoint Slice %s/%s port: %+v", epslNew.Namespace, epslNew.Name, *e.port)
			if err := p.deleteEndpoint(e.name, e.addr, e.port, eps); err != nil {
				klog.Errorf("failed to remove Endpoint Slice %s/%s port %+v with error: %+v", epslNew.Namespace, epslNew.Name, *e.port, err)
			}
			continue
		}
		if found && e.ready && !oldReady {
			// Case when Endpoint for port and address pair changed state from NOT Ready to Ready, so add a new port
			klog.V(5).Infof("adding Endpoint Slice %s/%s port: %+v", epslNew.Namespace, epslNew.Name, *e.port)
			if err := p.addEndpoint(e.name, e.addr, e.port); err != nil {
				klog.Errorf("failed to update Endpoint Slice %s/%s port %+v with error: %+v", epslNew.Namespace, epslNew.Name, *e.port, err)
			}
			continue
		}
		if found && !e.ready && !oldReady {
			// Case when nothing changed for port and address pair, ignoring it
			continue
		}
	}
	// Check for removed endpoint's ports, if found, remvoing all entries from EndpointMap
	info, _ = processEpSlice(storedEpSl)
	for _, e := range info {
		_, found := isPortInEndpointSlice(epslNew, e.port, e.addr)
		if !found && e.ready {
			// Case when Endpoint for port/address was in Ready state but then was deleted
			p.mu.Lock()
			eps, ok := p.endpointsMap[e.name]
			p.mu.Unlock()
			if !ok {
				continue
			}
			klog.V(5).Infof("removing Endpoint Slice %s/%s port: %+v", epslNew.Namespace, epslNew.Name, *e.port)
			if err := p.deleteEndpoint(e.name, e.addr, e.port, eps); err != nil {
				klog.Errorf("failed to remove Endpoint Slice %s/%s port %+v with error: %+v", epslNew.Namespace, epslNew.Name, *e.port, err)
				continue
			}
		}
	}
	p.cache.storeEpSlInCache(epslNew)
}
