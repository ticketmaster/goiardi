/*
 * Copyright (c) 2013-2018, Jeremy Bingham (<jeremy@goiardi.gl>)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package aclhelper is just an interface definition to allow access to the acl
// methods in various packages that can't import it directly because of import
// cycles.
package aclhelper

import (
	"github.com/casbin/casbin"
	"github.com/ctdk/goiardi/util"
)

type Member interface {
	IsACLRole() bool
	ACLName() string
	GetName() string
}

type Role interface {
	IsACLRole() bool
	ACLName() string
	GetName() string
	AllMembers() []Member
}

type Item interface {
	GetName() string
	ContainerKind() string
	ContainerType() string
}

// Pretty sure this will be useful in only one or two places, but so it goes.
type ACL struct {
	Perms map[string]*ACLItem
}

type ACLItem struct {
	Perm string
	Effect string
	Actors []string
	Groups []string
}

// Actor is an interface for objects that can make requests to the server. This
// is a duplicate of the Actor interface in github.com/ctdk/goiardi/actor.
type Actor interface {
	IsAdmin() bool
	IsValidator() bool
	IsSelf(interface{}) bool
	IsUser() bool
	IsClient() bool
	PublicKey() string
	SetPublicKey(interface{}) error
	GetName() string
	CheckPermEdit(map[string]interface{}, string) util.Gerror
	OrgName() string
	ACLName() string
	Authz() string
	IsACLRole() bool
}

type PermChecker interface {
	CheckItemPerm(Item, Actor, string) (bool, util.Gerror)
	RootCheckPerm(Actor, string) (bool, util.Gerror)
	EditItemPerm(Item, Member, []string, string) util.Gerror
	AddMembers(Role, []Member) error
	RemoveMembers(Role, []Member) error
	AddACLRole(Role) error
	RemoveACLRole(Role) error
	Enforcer() *casbin.SyncedEnforcer
	GetItemACL(Item) (*ACL, error)
}


func (a *ACL) ToJSON() map[string]map[string][]string {
	return nil
}