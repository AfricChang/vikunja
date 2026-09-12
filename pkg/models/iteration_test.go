// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.

package models

import (
	"testing"
	"time"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/user"
	"code.vikunja.io/api/pkg/web"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIterationLifecycle(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()
	u := &user.User{ID: 1}

	iteration := &Iteration{Title: "Custom cycle", StartDate: time.Date(2026, 8, 17, 0, 0, 0, 0, time.UTC), EndDate: time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC)}
	require.NoError(t, iteration.Create(s, u))
	assert.Equal(t, IterationStatusPlanned, iteration.Status)

	started, err := StartIteration(s, u, iteration.ID)
	require.NoError(t, err)
	assert.Equal(t, IterationStatusActive, started.Status)

	require.NoError(t, AddTasksToIteration(s, u, iteration.ID, []int64{3}, false))
	info, err := GetTaskIterationInfo(s, u, 3)
	require.NoError(t, err)
	require.NotNil(t, info.Current)
	assert.Equal(t, iteration.ID, info.Current.ID)
}

func TestIterationRejectsInvalidDates(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()
	err := (&Iteration{Title: "bad", StartDate: time.Now(), EndDate: time.Now().Add(-time.Hour)}).Create(s, &user.User{ID: 1})
	assert.Error(t, err)
}

func TestIterationPermissions(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()
	owner := &user.User{ID: 1}
	i := &Iteration{Title: "Private iteration", StartDate: time.Now(), EndDate: time.Now().Add(time.Hour)}
	require.NoError(t, i.Create(s, owner))

	for _, tt := range []struct {
		name string
		auth web.Auth
		want bool
	}{
		{"owner", owner, true},
		{"another user", &user.User{ID: 2}, false},
		{"link share with colliding id", &LinkSharing{ID: 1, ProjectID: 1, Permission: PermissionAdmin}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			candidate := &Iteration{ID: i.ID}
			can, _, err := candidate.CanRead(s, tt.auth)
			require.NoError(t, err)
			assert.Equal(t, tt.want, can)
			can, err = candidate.CanUpdate(s, tt.auth)
			require.NoError(t, err)
			assert.Equal(t, tt.want, can)
			can, err = candidate.CanDelete(s, tt.auth)
			require.NoError(t, err)
			assert.Equal(t, tt.want, can)
			_, _, err = ListIterationTasks(s, tt.auth, i.ID, "", "", "", "", "", 1, 20)
			if tt.want {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, ErrGenericForbidden{})
			}
		})
	}

	can, err := i.CanCreate(s, owner)
	require.NoError(t, err)
	assert.True(t, can)
	can, err = i.CanCreate(s, &LinkSharing{ID: 1, ProjectID: 1})
	require.NoError(t, err)
	assert.False(t, can)
}

func TestIterationTaskVisibilityAndSorting(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()
	owner := &user.User{ID: 1}
	i := &Iteration{Title: "Mixed project iteration", StartDate: time.Now(), EndDate: time.Now().Add(time.Hour)}
	require.NoError(t, i.Create(s, owner))
	parentID := int64(9)
	_, err := s.Insert(&Project{ID: 1000, Title: "Test5 inherited", OwnerID: 6, ParentProjectID: &parentID})
	require.NoError(t, err)

	tasks := []*Task{
		{ID: 1001, ProjectID: 1, Title: "D owned", CreatedByID: 1},
		{ID: 1002, ProjectID: 10, Title: "C direct share", CreatedByID: 6, Done: true},
		{ID: 1003, ProjectID: 1000, Title: "B inherited share", CreatedByID: 6},
		{ID: 1004, ProjectID: 6, Title: "A team share", CreatedByID: 6, Done: true},
		{ID: 1005, ProjectID: 2, Title: "Hidden project", CreatedByID: 3, Done: true},
	}
	for _, task := range tasks {
		task.Index = task.ID
		_, err := s.Insert(task)
		require.NoError(t, err)
		// Existing membership may survive revocation of the project's access.
		_, err = s.Insert(&IterationTask{IterationID: i.ID, TaskID: task.ID, AddedByID: owner.ID, AddedAt: time.Now()})
		require.NoError(t, err)
	}
	require.NoError(t, i.ReadOne(s, owner))
	assert.EqualValues(t, 4, i.TaskCount)
	assert.EqualValues(t, 2, i.DoneTaskCount)
	assert.InDelta(t, 0.5, i.CompletionRate, 0.001)

	for _, tt := range []struct {
		name    string
		sort    string
		order   string
		project string
		status  string
		page    int
		perPage int
		ids     []int64
		total   int64
	}{
		{"title ascending", "title", "asc", "", "", 1, 20, []int64{1004, 1003, 1002, 1001}, 4},
		{"project ascending", "project", "asc", "", "", 1, 20, []int64{1001, 1002, 1003, 1004}, 4},
		{"project descending", "project", "desc", "", "", 1, 20, []int64{1004, 1003, 1002, 1001}, 4},
		{"pagination", "title", "asc", "", "", 2, 2, []int64{1002, 1001}, 4},
		{"completed tasks", "id", "asc", "", "done", 1, 20, []int64{1002, 1004}, 2},
		{"inherited project filter", "id", "asc", "1000", "", 1, 20, []int64{1003}, 1},
		{"inaccessible project filter", "project", "asc", "2", "", 1, 20, []int64{}, 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			listed, total, err := ListIterationTasks(s, owner, i.ID, "", tt.project, tt.status, tt.sort, tt.order, tt.page, tt.perPage)
			require.NoError(t, err)
			ids := make([]int64, 0, len(listed))
			for _, task := range listed {
				ids = append(ids, task.ID)
			}
			assert.Equal(t, tt.ids, ids)
			assert.Equal(t, tt.total, total)
		})
	}

	_, err = s.Where("user_id = ? AND project_id = ?", owner.ID, 10).Delete(&ProjectUser{})
	require.NoError(t, err)
	require.NoError(t, i.ReadOne(s, owner))
	assert.EqualValues(t, 3, i.TaskCount)
	assert.EqualValues(t, 1, i.DoneTaskCount)
	listed, total, err := ListIterationTasks(s, owner, i.ID, "", "10", "", "title", "asc", 1, 20)
	require.NoError(t, err)
	assert.Empty(t, listed)
	assert.Zero(t, total)
}
