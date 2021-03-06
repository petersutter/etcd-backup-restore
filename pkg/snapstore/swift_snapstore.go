// Copyright (c) 2018 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
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

package snapstore

import (
	"fmt"
	"io"
	"path"
	"sort"
	"strings"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/objectstorage/v1/objects"
	"github.com/gophercloud/gophercloud/pagination"
)

// SwiftSnapStore is snapstore with Openstack Swift as backend
type SwiftSnapStore struct {
	SnapStore
	prefix string
	client *gophercloud.ServiceClient
	bucket string
}

// NewSwiftSnapStore create new SwiftSnapStore from shared configuration with specified bucket
func NewSwiftSnapStore(bucket, prefix string) (*SwiftSnapStore, error) {
	authOpts, err := openstack.AuthOptionsFromEnv()
	if err != nil {
		return nil, err
	}
	provider, err := openstack.AuthenticatedClient(authOpts)
	if err != nil {
		return nil, err

	}
	client, err := openstack.NewObjectStorageV1(provider, gophercloud.EndpointOpts{})
	if err != nil {
		return nil, err

	}
	return &SwiftSnapStore{
		prefix: prefix,
		client: client,
		bucket: bucket,
	}, nil

}

// Fetch should open reader for the snapshot file from store
func (s *SwiftSnapStore) Fetch(snap Snapshot) (io.ReadCloser, error) {
	resp := objects.Download(s.client, s.bucket, path.Join(s.prefix, snap.SnapDir, snap.SnapName), nil)
	return resp.Body, resp.Err
}

// Save will write the snapshot to store
func (s *SwiftSnapStore) Save(snap Snapshot, r io.Reader) error {
	opts := objects.CreateOpts{
		Content: r,
	}
	res := objects.Create(s.client, s.bucket, path.Join(s.prefix, snap.SnapDir, snap.SnapName), opts)
	return res.Err
}

// List will list the snapshots from store
func (s *SwiftSnapStore) List() (SnapList, error) {

	opts := &objects.ListOpts{
		Full:   false,
		Prefix: s.prefix,
	}
	// Retrieve a pager (i.e. a paginated collection)
	pager := objects.List(s.client, s.bucket, opts)
	var snapList SnapList
	// Define an anonymous function to be executed on each page's iteration
	err := pager.EachPage(func(page pagination.Page) (bool, error) {

		objectList, err := objects.ExtractNames(page)
		if err != nil {
			return false, err
		}
		for _, object := range objectList {
			name := strings.Replace(object, s.prefix+"/", "", 1)
			snap, err := ParseSnapshot(name)
			if err != nil {
				// Warning: the file can be a non snapshot file. Donot return error.
				fmt.Printf("Invalid snapshot found. Ignoring it:%s\n", name)
			} else {
				snapList = append(snapList, snap)
			}
		}
		return true, nil

	})
	if err != nil {
		return nil, err
	}

	sort.Sort(snapList)
	return snapList, nil

}

// Delete should delete the snapshot file from store
func (s *SwiftSnapStore) Delete(snap Snapshot) error {
	result := objects.Delete(s.client, s.bucket, path.Join(s.prefix, snap.SnapDir, snap.SnapName), nil)
	return result.Err
}
