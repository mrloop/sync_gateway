//  Copyright (c) 2012 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"

	"github.com/couchbaselabs/sync_gateway/base"
	ch "github.com/couchbaselabs/sync_gateway/channels"
)

/** A group that users can belong to, with associated channel permisisons. */
type roleImpl struct {
	Name_             string `json:"name,omitempty"`
	ExplicitChannels_ ch.Set `json:"admin_channels"`
	Channels_         ch.Set `json:"all_channels,omitempty"`
}

var kValidNameRegexp *regexp.Regexp

func init() {
	var err error
	kValidNameRegexp, err = regexp.Compile(`^[-+.@\w]*$`)
	if err != nil {
		panic("Bad kValidNameRegexp")
	}
}

func (role *roleImpl) initRole(name string, channels ch.Set) error {
	channels = channels.ExpandingStar()
	role.Name_ = name
	role.ExplicitChannels_ = channels
	return role.validate()
}

// Is this string a valid name for a User/Role? (Valid chars are alphanumeric and any of "_-+.@")
func IsValidPrincipalName(name string) bool {
	return kValidNameRegexp.MatchString(name)
}

// Creates a new Role object.
func (auth *Authenticator) NewRole(name string, channels ch.Set) (Role, error) {
	role := &roleImpl{}
	if err := role.initRole(name, channels); err != nil {
		return nil, err
	}
	if err := auth.rebuildChannels(role); err != nil {
		return nil, err
	}
	return role, nil
}

func (auth *Authenticator) UnmarshalRole(data []byte, defaultName string) (Role, error) {
	role := &roleImpl{}
	if err := json.Unmarshal(data, role); err != nil {
		return nil, err
	}
	if role.Name_ == "" {
		role.Name_ = defaultName
	}
	if err := role.validate(); err != nil {
		return nil, err
	}
	return role, nil
}

func docIDForRole(name string) string {
	return "role:" + name
}

func (role *roleImpl) docID() string {
	return docIDForRole(role.Name_)
}

// Key used in 'access' view (not same meaning as doc ID)
func (role *roleImpl) accessViewKey() string {
	return "role:" + role.Name_
}

//////// ACCESSORS:

func (role *roleImpl) Name() string {
	return role.Name_
}

func (role *roleImpl) Channels() ch.Set {
	return role.Channels_
}

func (role *roleImpl) setChannels(channels ch.Set) {
	role.Channels_ = channels
}

func (role *roleImpl) ExplicitChannels() ch.Set {
	return role.ExplicitChannels_
}

// Checks whether this role object contains valid data; if not, returns an error.
func (role *roleImpl) validate() error {
	if !IsValidPrincipalName(role.Name_) {
		return &base.HTTPError{http.StatusBadRequest, fmt.Sprintf("Invalid name %q", role.Name_)}
	}
	return role.ExplicitChannels_.Validate()
}

//////// CHANNEL AUTHORIZATION:

func (role *roleImpl) UnauthError(message string) error {
	if role.Name_ == "" {
		return &base.HTTPError{http.StatusUnauthorized, "login required"}
	}
	return &base.HTTPError{http.StatusForbidden, message}
}

// Returns true if the Role is allowed to access the channel.
// A nil Role means access control is disabled, so the function will return true.
func (role *roleImpl) CanSeeChannel(channel string) bool {
	return role == nil || role.Channels_.Contains(channel) || role.Channels_.Contains("*")
}

func (role *roleImpl) AuthorizeAllChannels(channels ch.Set) error {
	return authorizeAllChannels(role, channels)
}

// Returns an HTTP 403 error if the Role is not allowed to access all the given channels.
// A nil Role means access control is disabled, so the function will return nil.
func authorizeAllChannels(princ Principal, channels ch.Set) error {
	var forbidden []string
	for channel, _ := range channels {
		if !princ.CanSeeChannel(channel) {
			if forbidden == nil {
				forbidden = make([]string, 0, len(channels))
			}
			forbidden = append(forbidden, channel)
		}
	}
	if forbidden != nil {
		return princ.UnauthError(fmt.Sprintf("You are not allowed to see channels %v", forbidden))
	}
	return nil
}
