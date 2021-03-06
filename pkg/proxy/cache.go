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
	"sync"

	v1 "k8s.io/api/core/v1"
	discovery "k8s.io/api/discovery/v1beta1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
)

// cache defines a struct to store latest version of the seen service or endpoint. Once a service/endpoint add received
// and processed (ServicePorts are created), it will be added to cache map with key
// types.NamespacedName of a service/endpoint.
type cache struct {
	sync.Mutex
	svcCache  map[types.NamespacedName]*v1.Service
	epCache   map[types.NamespacedName]*v1.Endpoints
	epslCache map[types.NamespacedName]*discovery.EndpointSlice
}

// getCachedSvcVersion return version of stored service
func (c *cache) getCachedSvcVersion(name, namespace string) (string, error) {
	c.Lock()
	defer c.Unlock()
	s, ok := c.svcCache[types.NamespacedName{Name: name, Namespace: namespace}]
	if !ok {
		return "", fmt.Errorf("service %s/%s not found in the cache", namespace, name)
	}

	return s.ObjectMeta.GetResourceVersion(), nil
}

// getLastKnownSvcFromCache return pointer to the latest known/stored instance of the service
func (c *cache) getLastKnownSvcFromCache(name, namespace string) (*v1.Service, error) {
	c.Lock()
	defer c.Unlock()
	s, ok := c.svcCache[types.NamespacedName{Name: name, Namespace: namespace}]
	if !ok {
		return nil, fmt.Errorf("service %s/%s not found in the cache", namespace, name)
	}

	return s, nil
}

// storeSvcInCache stores in the cache instance of a service, if cache does not have already
// service, it will be added, if it already has, iy will be replaced with the one passed
// as a parameter.
func (c *cache) storeSvcInCache(s *v1.Service) {
	c.Lock()
	defer c.Unlock()
	c.svcCache[types.NamespacedName{Name: s.ObjectMeta.Name, Namespace: s.ObjectMeta.Namespace}] = s.DeepCopy()
}

// removeSvcFromCache removes stored service from cache.
func (c *cache) removeSvcFromCache(name, namespace string) {
	c.Lock()
	defer c.Unlock()
	if _, ok := c.svcCache[types.NamespacedName{Name: name, Namespace: namespace}]; ok {
		delete(c.svcCache, types.NamespacedName{Name: name, Namespace: namespace})
	} else {
		klog.Warningf("service %s/%s not found in the cache", namespace, name)
	}
}

// getCachedEpVersion return version of stored endpoint
func (c *cache) getCachedEpVersion(name, namespace string) (string, error) {
	c.Lock()
	defer c.Unlock()
	ep, ok := c.epCache[types.NamespacedName{Name: name, Namespace: namespace}]
	if !ok {
		return "", fmt.Errorf("endpoint %s/%s not found in the cache", namespace, name)
	}

	return ep.ObjectMeta.GetResourceVersion(), nil
}

// getLastKnownEpFromCache return pointer to the latest known/stored instance of the endpoint
func (c *cache) getLastKnownEpFromCache(name, namespace string) (*v1.Endpoints, error) {
	c.Lock()
	defer c.Unlock()
	ep, ok := c.epCache[types.NamespacedName{Name: name, Namespace: namespace}]
	if !ok {
		return nil, fmt.Errorf("endpoint %s/%s not found in the cache", namespace, name)
	}

	return ep, nil
}

// storeEpInCache stores in the cache instance of a endpoint, if cache does not have already
// endpoint, it will be added, if it already has, iy will be replaced with the one passed
// as a parameter.
func (c *cache) storeEpInCache(ep *v1.Endpoints) {
	c.Lock()
	defer c.Unlock()
	c.epCache[types.NamespacedName{Name: ep.ObjectMeta.Name, Namespace: ep.ObjectMeta.Namespace}] = ep.DeepCopy()
}

// removeEpFromCache removes stored service from cache.
func (c *cache) removeEpFromCache(name, namespace string) {
	c.Lock()
	defer c.Unlock()
	if _, ok := c.epCache[types.NamespacedName{Name: name, Namespace: namespace}]; ok {
		delete(c.epCache, types.NamespacedName{Name: name, Namespace: namespace})
	} else {
		klog.Warningf("endpoint %s/%s not found in the cache", namespace, name)
	}
}

// getCachedEpSlVersion return version of stored endpoint slice
func (c *cache) getCachedEpSlVersion(name, namespace string) (string, error) {
	c.Lock()
	defer c.Unlock()
	epsl, ok := c.epslCache[types.NamespacedName{Name: name, Namespace: namespace}]
	if !ok {
		return "", fmt.Errorf("endpoint slice %s/%s not found in the cache", namespace, name)
	}

	return epsl.ObjectMeta.GetResourceVersion(), nil
}

// getLastKnownEpSlFromCache return pointer to the latest known/stored instance of the endpoint slice
func (c *cache) getLastKnownEpSlFromCache(name, namespace string) (*discovery.EndpointSlice, error) {
	c.Lock()
	defer c.Unlock()
	epsl, ok := c.epslCache[types.NamespacedName{Name: name, Namespace: namespace}]
	if !ok {
		return nil, fmt.Errorf("endpoint slice %s/%s not found in the cache", namespace, name)
	}

	return epsl, nil
}

// storeEpSlInCache stores in the cache instance of a endpoint slice, if cache does not have already
// endpoint slice, it will be added, if it already has, iy will be replaced with the one passed
// as a parameter.
func (c *cache) storeEpSlInCache(epsl *discovery.EndpointSlice) {
	c.Lock()
	defer c.Unlock()
	c.epslCache[types.NamespacedName{Name: epsl.ObjectMeta.Name, Namespace: epsl.ObjectMeta.Namespace}] = epsl.DeepCopy()
}

// removeEpSlFromCache removes stored service from cache.
func (c *cache) removeEpSlFromCache(name, namespace string) {
	c.Lock()
	defer c.Unlock()
	if _, ok := c.epslCache[types.NamespacedName{Name: name, Namespace: namespace}]; ok {
		delete(c.epslCache, types.NamespacedName{Name: name, Namespace: namespace})
	} else {
		klog.Warningf("endpoint slice %s/%s not found in the cache", namespace, name)
	}
}
