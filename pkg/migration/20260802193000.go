// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.

package migration

import (
	"time"

	"src.techknowlogick.com/xormigrate"
	"xorm.io/xorm"
)

type iteration20260802193000 struct {
	ID                  int64      `xorm:"bigint autoincr not null unique pk"`
	Title               string     `xorm:"varchar(250) not null"`
	Description         string     `xorm:"longtext null"`
	StartDate           time.Time  `xorm:"DATETIME not null INDEX 'start_date'"`
	EndDate             time.Time  `xorm:"DATETIME not null INDEX 'end_date'"`
	Status              string     `xorm:"varchar(20) not null INDEX default 'planned'"`
	OwnerID             int64      `xorm:"bigint not null INDEX 'owner_id'"`
	PreviousIterationID int64      `xorm:"bigint null INDEX 'previous_iteration_id'"`
	ClosedAt            *time.Time `xorm:"DATETIME null INDEX 'closed_at'"`
	ClosedByID          int64      `xorm:"bigint null 'closed_by_id'"`
	Created             time.Time  `xorm:"created not null"`
	Updated             time.Time  `xorm:"updated not null"`
}

func (iteration20260802193000) TableName() string { return "iterations" }

type iterationTask20260802193000 struct {
	ID                int64     `xorm:"bigint autoincr not null unique pk"`
	IterationID       int64     `xorm:"bigint not null INDEX unique(iteration_task) 'iteration_id'"`
	TaskID            int64     `xorm:"bigint not null unique INDEX unique(iteration_task) 'task_id'"`
	AddedAt           time.Time `xorm:"DATETIME not null 'added_at'"`
	AddedByID         int64     `xorm:"bigint not null 'added_by_id'"`
	SourceIterationID int64     `xorm:"bigint null INDEX 'source_iteration_id'"`
}

func (iterationTask20260802193000) TableName() string { return "iteration_tasks" }

type taskIterationHistory20260802193000 struct {
	ID                int64      `xorm:"bigint autoincr not null unique pk"`
	TaskID            int64      `xorm:"bigint not null INDEX 'task_id'"`
	IterationID       int64      `xorm:"bigint not null INDEX 'iteration_id'"`
	JoinedAt          time.Time  `xorm:"DATETIME not null INDEX 'joined_at'"`
	JoinedByID        int64      `xorm:"bigint not null 'joined_by_id'"`
	LeftAt            *time.Time `xorm:"DATETIME null INDEX 'left_at'"`
	LeftByID          int64      `xorm:"bigint null 'left_by_id'"`
	LeftReason        string     `xorm:"varchar(24) null 'left_reason'"`
	SourceIterationID int64      `xorm:"bigint null 'source_iteration_id'"`
	TargetIterationID int64      `xorm:"bigint null 'target_iteration_id'"`
	TaskTitle         string     `xorm:"varchar(250) not null 'task_title'"`
	ProjectID         int64      `xorm:"bigint not null INDEX 'project_id'"`
}

func (taskIterationHistory20260802193000) TableName() string { return "task_iteration_history" }

func init() {
	migrations = append(migrations, &xormigrate.Migration{
		ID:          "20260802193000",
		Description: "Add iterations and task rollover history",
		Migrate: func(tx *xorm.Engine) error {
			return tx.CreateTables(iteration20260802193000{}, iterationTask20260802193000{}, taskIterationHistory20260802193000{})
		},
		Rollback: func(tx *xorm.Engine) error { return nil },
	})
}
