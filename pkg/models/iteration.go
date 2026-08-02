// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package models

import (
	"strings"
	"time"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/user"
	"code.vikunja.io/api/pkg/web"

	"xorm.io/builder"
	"xorm.io/xorm"
)

type IterationStatus string

const (
	IterationStatusPlanned IterationStatus = "planned"
	IterationStatusActive  IterationStatus = "active"
	IterationStatusClosed  IterationStatus = "closed"
)

type Iteration struct {
	ID                  int64           `xorm:"bigint autoincr not null unique pk" json:"id" param:"iteration" readOnly:"true" doc:"The unique numeric id of this iteration."`
	Title               string          `xorm:"varchar(250) not null" json:"title" valid:"required,runelength(1|250)" minLength:"1" maxLength:"250" doc:"The iteration name."`
	Description         string          `xorm:"longtext null" json:"description" doc:"The iteration goal or notes."`
	StartDate           time.Time       `xorm:"DATETIME not null INDEX 'start_date'" json:"start_date" doc:"When the iteration starts."`
	EndDate             time.Time       `xorm:"DATETIME not null INDEX 'end_date'" json:"end_date" doc:"When the iteration ends. Must be after start_date."`
	Status              IterationStatus `xorm:"varchar(20) not null INDEX default 'planned'" json:"status" readOnly:"true" doc:"planned, active, or closed."`
	OwnerID             int64           `xorm:"bigint not null INDEX 'owner_id'" json:"-"`
	PreviousIterationID int64           `xorm:"bigint null INDEX 'previous_iteration_id'" json:"previous_iteration_id" doc:"The previous iteration in this planning chain, or 0."`
	ClosedAt            *time.Time      `xorm:"DATETIME null INDEX 'closed_at'" json:"closed_at" readOnly:"true"`
	ClosedByID          int64           `xorm:"bigint null 'closed_by_id'" json:"closed_by_id" readOnly:"true"`
	Created             time.Time       `xorm:"created not null" json:"created" readOnly:"true"`
	Updated             time.Time       `xorm:"updated not null" json:"updated" readOnly:"true"`

	Owner            *user.User `xorm:"-" json:"owner,omitempty" readOnly:"true"`
	TaskCount        int64      `xorm:"-" json:"task_count" readOnly:"true"`
	DoneTaskCount    int64      `xorm:"-" json:"done_task_count" readOnly:"true"`
	HiddenTaskCount  int64      `xorm:"-" json:"hidden_task_count" readOnly:"true"`
	CompletionRate   float64    `xorm:"-" json:"completion_rate" readOnly:"true"`
	HasNextIteration bool       `xorm:"-" json:"has_next_iteration" readOnly:"true"`
	Overlapping      bool       `xorm:"-" json:"overlapping" readOnly:"true"`

	SortBy       string          `xorm:"-" json:"-"`
	OrderBy      string          `xorm:"-" json:"-"`
	FilterStatus IterationStatus `xorm:"-" json:"-"`

	web.CRUDable    `xorm:"-" json:"-"`
	web.Permissions `xorm:"-" json:"-"`
}

func (*Iteration) TableName() string { return "iterations" }

type IterationTask struct {
	ID                int64     `xorm:"bigint autoincr not null unique pk" json:"id" readOnly:"true"`
	IterationID       int64     `xorm:"bigint not null INDEX unique(iteration_task) 'iteration_id'" json:"iteration_id"`
	TaskID            int64     `xorm:"bigint not null unique INDEX unique(iteration_task) 'task_id'" json:"task_id"`
	AddedAt           time.Time `xorm:"DATETIME not null 'added_at'" json:"added_at" readOnly:"true"`
	AddedByID         int64     `xorm:"bigint not null 'added_by_id'" json:"added_by_id" readOnly:"true"`
	SourceIterationID int64     `xorm:"bigint null INDEX 'source_iteration_id'" json:"source_iteration_id" readOnly:"true"`
}

func (*IterationTask) TableName() string { return "iteration_tasks" }

type TaskIterationHistory struct {
	ID                int64           `xorm:"bigint autoincr not null unique pk" json:"id" readOnly:"true"`
	TaskID            int64           `xorm:"bigint not null INDEX 'task_id'" json:"task_id"`
	IterationID       int64           `xorm:"bigint not null INDEX 'iteration_id'" json:"iteration_id"`
	JoinedAt          time.Time       `xorm:"DATETIME not null INDEX 'joined_at'" json:"joined_at"`
	JoinedByID        int64           `xorm:"bigint not null 'joined_by_id'" json:"joined_by_id"`
	LeftAt            *time.Time      `xorm:"DATETIME null INDEX 'left_at'" json:"left_at"`
	LeftByID          int64           `xorm:"bigint null 'left_by_id'" json:"left_by_id"`
	LeftReason        string          `xorm:"varchar(24) null 'left_reason'" json:"left_reason"`
	SourceIterationID int64           `xorm:"bigint null 'source_iteration_id'" json:"source_iteration_id"`
	TargetIterationID int64           `xorm:"bigint null 'target_iteration_id'" json:"target_iteration_id"`
	TaskTitle         string          `xorm:"varchar(250) not null 'task_title'" json:"task_title"`
	ProjectID         int64           `xorm:"bigint not null INDEX 'project_id'" json:"project_id"`
	IterationTitle    string          `xorm:"-" json:"iteration_title" readOnly:"true"`
	IterationStatus   IterationStatus `xorm:"-" json:"iteration_status" readOnly:"true"`
}

func (*TaskIterationHistory) TableName() string { return "task_iteration_history" }

func (i *Iteration) validate() error {
	if i.StartDate.IsZero() || i.EndDate.IsZero() || !i.EndDate.After(i.StartDate) {
		return ErrInvalidIterationDates{IterationID: i.ID}
	}
	return nil
}

func (i *Iteration) Create(s *xorm.Session, a web.Auth) error {
	if err := i.validate(); err != nil {
		return err
	}
	if _, isShare := a.(*LinkSharing); isShare {
		return ErrGenericForbidden{}
	}
	if i.PreviousIterationID > 0 {
		previous, err := GetIterationByID(s, i.PreviousIterationID)
		if err != nil {
			return err
		}
		if previous.OwnerID != a.GetID() {
			return ErrGenericForbidden{}
		}
	}
	i.ID = 0
	i.OwnerID = a.GetID()
	i.Status = IterationStatusPlanned
	i.ClosedAt = nil
	i.ClosedByID = 0
	if _, err := s.Insert(i); err != nil {
		return err
	}
	return hydrateIteration(s, i, a)
}

func GetIterationByID(s *xorm.Session, id int64) (*Iteration, error) {
	i := &Iteration{}
	has, err := s.Where("id = ?", id).Get(i)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, ErrIterationDoesNotExist{IterationID: id}
	}
	return i, nil
}

func (i *Iteration) ReadOne(s *xorm.Session, a web.Auth) error { return hydrateIteration(s, i, a) }

func (i *Iteration) ReadAll(s *xorm.Session, a web.Auth, search string, page, perPage int) (any, int, int64, error) {
	if _, isShare := a.(*LinkSharing); isShare {
		return []*Iteration{}, 0, 0, nil
	}
	var condition builder.Cond = builder.Eq{"owner_id": a.GetID()}
	if i.FilterStatus != "" {
		condition = builder.And(condition, builder.Eq{"status": i.FilterStatus})
	}
	if search != "" {
		condition = builder.And(condition, db.MultiFieldSearch([]string{"title", "description"}, search))
	}
	total, err := s.Where(condition).Count(&Iteration{})
	if err != nil {
		return nil, 0, 0, err
	}
	sortBy := i.SortBy
	if sortBy != "end_date" && sortBy != "title" && sortBy != "created" {
		sortBy = "start_date"
	}
	order := strings.ToUpper(i.OrderBy)
	if order != "ASC" {
		order = "DESC"
	}
	items := []*Iteration{}
	if err := s.Where(condition).OrderBy(sortBy + " " + order).Limit(getLimitFromPageIndex(page, perPage)).Find(&items); err != nil {
		return nil, 0, 0, err
	}
	for _, item := range items {
		if err := hydrateIteration(s, item, a); err != nil {
			return nil, 0, 0, err
		}
	}
	return items, len(items), total, nil
}

func (i *Iteration) Update(s *xorm.Session, a web.Auth) error {
	existing, err := GetIterationByID(s, i.ID)
	if err != nil {
		return err
	}
	if existing.Status == IterationStatusClosed {
		return ErrIterationClosed{IterationID: i.ID}
	}
	if err := i.validate(); err != nil {
		return err
	}
	i.OwnerID = existing.OwnerID
	i.Status = existing.Status
	_, err = s.Where("id = ?", i.ID).Cols("title", "description", "start_date", "end_date", "previous_iteration_id").Update(i)
	if err != nil {
		return err
	}
	return hydrateIteration(s, i, a)
}

func (i *Iteration) Delete(s *xorm.Session, _ web.Auth) error {
	count, err := s.Where("iteration_id = ?", i.ID).Count(&TaskIterationHistory{})
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrIterationHasHistory{IterationID: i.ID}
	}
	_, err = s.Where("id = ? AND status = ?", i.ID, IterationStatusPlanned).Delete(&Iteration{})
	return err
}

func hydrateIteration(s *xorm.Session, i *Iteration, a web.Auth) error {
	owner, err := user.GetUserByID(s, i.OwnerID)
	if err != nil {
		return err
	}
	i.Owner = owner
	visible := accessibleProjectIDsSubquery(a, "tasks.project_id")
	type counts struct {
		Total int64 `xorm:"total"`
		Done  int64 `xorm:"done"`
	}
	c := &counts{}
	_, err = s.Table("iteration_tasks").
		Select("COUNT(tasks.id) AS total, COALESCE(SUM(CASE WHEN tasks.done = 1 THEN 1 ELSE 0 END), 0) AS done").
		Join("INNER", "tasks", "tasks.id = iteration_tasks.task_id").
		Where(builder.And(builder.Eq{"iteration_tasks.iteration_id": i.ID}, visible, taskNotDeletedCond("tasks"))).Get(c)
	if err != nil {
		return err
	}
	i.TaskCount, i.DoneTaskCount = c.Total, c.Done
	if i.TaskCount > 0 {
		i.CompletionRate = float64(i.DoneTaskCount) / float64(i.TaskCount)
	}
	i.HiddenTaskCount = 0
	hasNext, err := s.Where("previous_iteration_id = ?", i.ID).Exist(&Iteration{})
	if err != nil {
		return err
	}
	i.HasNextIteration = hasNext
	i.Overlapping, err = s.Where("owner_id = ? AND id <> ? AND start_date < ? AND end_date > ?", i.OwnerID, i.ID, i.EndDate, i.StartDate).Exist(&Iteration{})
	return err
}

func ListIterationTasks(s *xorm.Session, a web.Auth, iterationID int64, search, projectFilter, statusFilter, sortBy, orderBy string, page, perPage int) ([]*Task, int64, error) {
	if _, err := requireOwnedIteration(s, a, iterationID); err != nil {
		return nil, 0, err
	}
	cond := builder.And(
		builder.Eq{"iteration_tasks.iteration_id": iterationID},
		accessibleProjectIDsSubquery(a, "tasks.project_id"),
		taskNotDeletedCond("tasks"),
	)
	if search != "" {
		cond = cond.And(db.MultiFieldSearch([]string{"tasks.title", "tasks.description"}, search))
	}
	if projectFilter != "" && projectFilter != "0" {
		cond = cond.And(builder.Eq{"tasks.project_id": projectFilter})
	}
	if statusFilter == "done" {
		cond = cond.And(builder.Eq{"tasks.done": true})
	}
	if statusFilter == "open" {
		cond = cond.And(builder.Eq{"tasks.done": false})
	}
	total, err := s.Table("tasks").Join("INNER", "iteration_tasks", "iteration_tasks.task_id = tasks.id").Where(cond).Count(&Task{})
	if err != nil {
		return nil, 0, err
	}
	allowedSort := map[string]string{"title": "tasks.title", "done": "tasks.done", "priority": "tasks.priority", "due_date": "tasks.due_date", "start_date": "tasks.start_date", "id": "tasks.id"}
	column := allowedSort[sortBy]
	if column == "" {
		column = "tasks.done"
	}
	direction := strings.ToUpper(orderBy)
	if direction != "DESC" {
		direction = "ASC"
	}
	tasks := []*Task{}
	err = s.Table("tasks").Join("INNER", "iteration_tasks", "iteration_tasks.task_id = tasks.id").Where(cond).
		OrderBy(column + " " + direction + ", tasks.id DESC").Limit(getLimitFromPageIndex(page, perPage)).Find(&tasks)
	if err != nil {
		return nil, 0, err
	}
	taskMap := make(map[int64]*Task, len(tasks))
	for _, task := range tasks {
		taskMap[task.ID] = task
	}
	if err := addMoreInfoToTasks(s, taskMap, a, nil, nil); err != nil {
		return nil, 0, err
	}
	return tasks, total, nil
}

func (i *Iteration) CanCreate(_ *xorm.Session, a web.Auth) (bool, error) {
	_, isShare := a.(*LinkSharing)
	return !isShare, nil
}

func (i *Iteration) CanRead(s *xorm.Session, a web.Auth) (bool, int, error) {
	if _, isShare := a.(*LinkSharing); isShare {
		return false, 0, nil
	}
	loaded, err := GetIterationByID(s, i.ID)
	if err != nil {
		return false, 0, err
	}
	*i = *loaded
	return i.OwnerID == a.GetID(), int(PermissionAdmin), nil
}

func (i *Iteration) CanUpdate(s *xorm.Session, a web.Auth) (bool, error) {
	loaded, err := GetIterationByID(s, i.ID)
	if err != nil {
		return false, err
	}
	return loaded.OwnerID == a.GetID(), nil
}

func (i *Iteration) CanDelete(s *xorm.Session, a web.Auth) (bool, error) {
	loaded, err := GetIterationByID(s, i.ID)
	if err != nil {
		return false, err
	}
	return loaded.OwnerID == a.GetID() && loaded.Status == IterationStatusPlanned, nil
}
