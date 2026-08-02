// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.

package models

import (
	"testing"
	"time"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/user"

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
