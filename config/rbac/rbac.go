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
	_ "embed"

	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/yaml"
)

//go:embed role.yaml
var contentClusterRole []byte

//go:embed namespaced_role.yaml
var contentNamespacedRole []byte

// ManagerClusterRole returns the operator's cluster-scoped RBAC role.
func ManagerClusterRole() *rbacv1.ClusterRole {
	var cr rbacv1.ClusterRole
	if err := yaml.Unmarshal(contentClusterRole, &cr); err != nil {
		panic("failed to parse role.yaml: " + err.Error())
	}

	return cr.DeepCopy()
}

// ManagerNamespacedRole returns the operator's namespace-scoped RBAC role.
func ManagerNamespacedRole() *rbacv1.Role {
	var r rbacv1.Role
	if err := yaml.Unmarshal(contentNamespacedRole, &r); err != nil {
		panic("failed to parse namespaced_role.yaml: " + err.Error())
	}

	return r.DeepCopy()
}
