// Copyright 2026 Intel Corporation. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package rbac

import (
	"testing"

	rbacv1 "k8s.io/api/rbac/v1"
)

// namespacedOnlyResources are core-API resources that must only appear in the
// namespace-scoped Role, never in the ClusterRole.  Granting these at cluster
// scope would give the operator read/write access to every namespace.
var namespacedOnlyResources = []string{
	"secrets",
	"configmaps",
	"services",
	"serviceaccounts",
	"daemonsets",
	"deployments",
	"jobs",
	"pods/log",
}

// secretWriteVerbs are verbs that must not be granted on secrets even inside
// the namespace-scoped Role; the operator only needs to read image pull secrets.
var secretWriteVerbs = []string{
	"create",
	"update",
	"patch",
	"delete",
	"deletecollection",
}

func parseClusterRole(t *testing.T) rbacv1.ClusterRole {
	t.Helper()

	return *ManagerClusterRole()
}

func parseRole(t *testing.T) rbacv1.Role {
	t.Helper()

	return *ManagerNamespacedRole()
}

// containsString returns true if slice contains s or the wildcard "*".
func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s || v == "*" {
			return true
		}
	}

	return false
}

// TestClusterRole_NoWildcards ensures the operator ClusterRole never uses
// wildcard verbs, resources, or API groups, which would grant unbounded access.
func TestClusterRole_NoWildcards(t *testing.T) {
	cr := parseClusterRole(t)

	for _, rule := range cr.Rules {
		for _, apiGroup := range rule.APIGroups {
			if apiGroup == "*" {
				t.Errorf("ClusterRole rule has wildcard apiGroup: %+v", rule)
			}
		}

		for _, resource := range rule.Resources {
			if resource == "*" {
				t.Errorf("ClusterRole rule has wildcard resource: %+v", rule)
			}
		}

		for _, verb := range rule.Verbs {
			if verb == "*" {
				t.Errorf("ClusterRole rule has wildcard verb: %+v", rule)
			}
		}
	}
}

// TestClusterRole_NoNamespacedScopedResources verifies that resources which
// should only be accessible within the operator's own namespace are not
// granted at cluster scope.  A ClusterRole rule for e.g. "secrets" would
// allow the operator to read secrets from every namespace in the cluster.
func TestClusterRole_NoNamespacedScopedResources(t *testing.T) {
	cr := parseClusterRole(t)

	for _, rule := range cr.Rules {
		for _, resource := range rule.Resources {
			for _, forbidden := range namespacedOnlyResources {
				if resource == forbidden {
					t.Errorf("ClusterRole must not grant access to %q (cluster-wide); "+
						"use the namespaced Role instead. Offending rule: %+v", forbidden, rule)
				}
			}
		}
	}
}

// TestNamespacedRole_SecretsReadOnly verifies that the namespace-scoped Role
// grants only read access to secrets.  The operator reads image pull secrets
// but must never create, modify, or delete them.
func TestNamespacedRole_SecretsReadOnly(t *testing.T) {
	r := parseRole(t)

	for _, rule := range r.Rules {
		if !containsString(rule.Resources, "secrets") {
			continue
		}

		for _, verb := range rule.Verbs {
			for _, writeVerb := range secretWriteVerbs {
				if verb == writeVerb {
					t.Errorf("namespaced Role must not grant %q on secrets; "+
						"only get/list are allowed. Offending rule: %+v", verb, rule)
				}
			}
		}
	}
}

// TestNamespacedRole_NoWildcards mirrors the ClusterRole wildcard check for
// the namespace-scoped Role.
func TestNamespacedRole_NoWildcards(t *testing.T) {
	r := parseRole(t)

	for _, rule := range r.Rules {
		for _, apiGroup := range rule.APIGroups {
			if apiGroup == "*" {
				t.Errorf("namespaced Role rule has wildcard apiGroup: %+v", rule)
			}
		}

		for _, resource := range rule.Resources {
			if resource == "*" {
				t.Errorf("namespaced Role rule has wildcard resource: %+v", rule)
			}
		}

		for _, verb := range rule.Verbs {
			if verb == "*" {
				t.Errorf("namespaced Role rule has wildcard verb: %+v", rule)
			}
		}
	}
}
