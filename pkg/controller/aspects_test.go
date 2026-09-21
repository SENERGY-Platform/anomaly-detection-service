/*
 * Copyright 2026 InfAI (CC SES)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package controller

import (
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/SENERGY-Platform/anomaly-detection-service/pkg/configuration"
	"github.com/SENERGY-Platform/anomaly-detection-service/pkg/handler"
	devicerepo "github.com/SENERGY-Platform/device-repository/v2/lib/client"
	"github.com/SENERGY-Platform/models/go/models"
)

const (
	aspectA    = models.URN_PREFIX + "aspect:a"
	aspectASub = models.URN_PREFIX + "aspect:a-sub"
	aspectB    = models.URN_PREFIX + "aspect:b"
	aspectC    = models.URN_PREFIX + "aspect:c"
)

// aspectNodeRepo answers aspect node queries and nothing else. The embedded interface is nil
// on purpose: a call to any other method panics instead of quietly returning a zero value.
// The device-repository test client cannot stand in here, its in-memory database has no
// aspect writes.
type aspectNodeRepo struct {
	devicerepo.Interface
	nodes     map[string]models.AspectNode
	requested [][]string
	err       error
}

func newAspectNodeRepo() *aspectNodeRepo {
	return &aspectNodeRepo{
		nodes: map[string]models.AspectNode{
			aspectA: {Id: aspectA, Name: "a", ChildIds: []string{aspectASub}, DescendentIds: []string{aspectASub}},
			aspectB: {Id: aspectB, Name: "b"},
			aspectC: {Id: aspectC, Name: "c"},
		},
	}
}

func (this *aspectNodeRepo) ListAspectNodes(options devicerepo.AspectListOptions) (result []models.AspectNode, total int64, err error, code int) {
	this.requested = append(this.requested, options.Ids)
	if this.err != nil {
		return nil, 0, this.err, 500
	}
	for _, id := range options.Ids {
		if node, ok := this.nodes[id]; ok {
			result = append(result, node)
		}
	}
	return result, int64(len(result)), nil, 200
}

func newAspectTestController(repo devicerepo.Interface) *Controller {
	return &Controller{
		config:           configuration.Config{},
		deviceRepoClient: repo,
	}
}

func TestGetAspectNodes(t *testing.T) {
	t.Run("single aspect", func(t *testing.T) {
		repo := newAspectNodeRepo()
		nodes, err := newAspectTestController(repo).getAspectNodes([]string{aspectA})
		if err != nil {
			t.Error(err)
			return
		}
		if !reflect.DeepEqual(nodeIds(nodes), []string{aspectA}) {
			t.Errorf("unexpected nodes %#v", nodeIds(nodes))
		}
	})

	t.Run("aspect list is requested in one call", func(t *testing.T) {
		repo := newAspectNodeRepo()
		nodes, err := newAspectTestController(repo).getAspectNodes([]string{aspectA, aspectB})
		if err != nil {
			t.Error(err)
			return
		}
		if !reflect.DeepEqual(nodeIds(nodes), []string{aspectA, aspectB}) {
			t.Errorf("unexpected nodes %#v", nodeIds(nodes))
		}
		expectedRequests := [][]string{{aspectA, aspectB}}
		if !reflect.DeepEqual(repo.requested, expectedRequests) {
			t.Errorf("unexpected requests\ne=%#v\na=%#v\n", expectedRequests, repo.requested)
		}
	})

	// The subtree ids have to survive: they are what lets an aspect cover its descendants, so
	// a handler registered for a parent aspect still matches a value carrying a child of it.
	t.Run("subtree ids are kept", func(t *testing.T) {
		repo := newAspectNodeRepo()
		nodes, err := newAspectTestController(repo).getAspectNodes([]string{aspectA})
		if err != nil {
			t.Error(err)
			return
		}
		if len(nodes) != 1 {
			t.Errorf("expected 1 node, got %v", len(nodes))
			return
		}
		if !slices.Contains(nodes[0].DescendentIds, aspectASub) {
			t.Errorf("expected %v in descendent ids, got %#v", aspectASub, nodes[0].DescendentIds)
		}
	})

	t.Run("no aspect is no filter", func(t *testing.T) {
		repo := newAspectNodeRepo()
		nodes, err := newAspectTestController(repo).getAspectNodes(nil)
		if err != nil {
			t.Error(err)
			return
		}
		if len(nodes) != 0 {
			t.Errorf("expected no nodes, got %#v", nodes)
		}
		if len(repo.requested) != 0 {
			t.Errorf("expected no request, got %#v", repo.requested)
		}
	})

	// An unknown aspect has to fail loudly. Passing the short answer on would leave the
	// handler registered and silently never matching anything.
	t.Run("unknown aspect is an error", func(t *testing.T) {
		repo := newAspectNodeRepo()
		_, err := newAspectTestController(repo).getAspectNodes([]string{aspectA, models.URN_PREFIX + "aspect:unknown"})
		if err == nil {
			t.Error("expected an error for an unknown aspect id")
		}
	})

	t.Run("repository error is passed on", func(t *testing.T) {
		repo := newAspectNodeRepo()
		repo.err = errors.New("nope")
		_, err := newAspectTestController(repo).getAspectNodes([]string{aspectA})
		if !errors.Is(err, repo.err) {
			t.Errorf("expected the repository error, got %#v", err)
		}
	})
}

// TestCreateRouterEntryAspectNodes checks the alias one level up: a handler entry built the
// deprecated way and one built with a one element list ask for the same aspect nodes.
func TestCreateRouterEntryAspectNodes(t *testing.T) {
	tests := []struct {
		name     string
		entry    handler.Entry
		expected []string
	}{
		{
			name:     "deprecated single aspect",
			entry:    handler.Entry{Name: "deprecated", Aspect: aspectA},
			expected: []string{aspectA},
		},
		{
			name:     "list with one element",
			entry:    handler.Entry{Name: "list", Aspects: []string{aspectA}},
			expected: []string{aspectA},
		},
		{
			name:     "list with two elements",
			entry:    handler.Entry{Name: "list2", Aspects: []string{aspectA, aspectB}},
			expected: []string{aspectA, aspectB},
		},
		{
			name:     "no aspect",
			entry:    handler.Entry{Name: "none"},
			expected: nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := newAspectNodeRepo()
			info, err := newAspectTestController(repo).createRouterEntry(test.entry, nil, nil)
			if err != nil {
				t.Error(err)
				return
			}
			if !reflect.DeepEqual(nodeIds(info.aspectNodes), test.expected) {
				t.Errorf("unexpected nodes\ne=%#v\na=%#v\n", test.expected, nodeIds(info.aspectNodes))
			}
		})
	}
}

func nodeIds(nodes []models.AspectNode) (result []string) {
	for _, node := range nodes {
		result = append(result, node.Id)
	}
	return result
}
