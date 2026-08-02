// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.

package models

import (
	"time"

	"code.vikunja.io/api/pkg/web"
	"xorm.io/xorm"
)

const (
	IterationLeftRemoved     = "removed"
	IterationLeftTransferred = "transferred"
	IterationLeftClosed      = "iteration_closed"
)

type IterationTaskChange struct {
	TaskIDs         []int64 `json:"task_ids"`
	ConfirmTransfer bool    `json:"confirm_transfer"`
}

type IterationMoveTasks struct {
	TaskIDs           []int64 `json:"task_ids"`
	TargetIterationID int64   `json:"target_iteration_id"`
}

type IterationCloseRequest struct {
	RolloverTaskIDs   []int64 `json:"rollover_task_ids"`
	TargetIterationID int64   `json:"target_iteration_id"`
}

type IterationCloseResult struct {
	Iteration               *Iteration `json:"iteration"`
	TargetIteration         *Iteration `json:"target_iteration,omitempty"`
	RolledOverTaskIDs       []int64    `json:"rolled_over_task_ids"`
	SkippedCompletedTaskIDs []int64    `json:"skipped_completed_task_ids"`
}

type TaskIterationInfo struct {
	Current *Iteration              `json:"current,omitempty"`
	History []*TaskIterationHistory `json:"history"`
}

func requireOwnedIteration(s *xorm.Session, a web.Auth, id int64) (*Iteration, error) {
	if _, isShare := a.(*LinkSharing); isShare {
		return nil, ErrGenericForbidden{}
	}
	i, err := GetIterationByID(s, id)
	if err != nil {
		return nil, err
	}
	if i.OwnerID != a.GetID() {
		return nil, ErrGenericForbidden{}
	}
	return i, nil
}

func checkTaskWriteAccess(s *xorm.Session, a web.Auth, taskIDs []int64) (map[int64]*Task, error) {
	if len(taskIDs) == 0 {
		return nil, ErrIterationTasksNeedAtLeastOne{}
	}
	tasks, err := GetTasksSimpleByIDs(s, taskIDs)
	if err != nil {
		return nil, err
	}
	if len(tasks) != len(uniqueInt64s(taskIDs)) {
		return nil, ErrTaskDoesNotExist{}
	}
	result := make(map[int64]*Task, len(tasks))
	projects := map[int64]struct{}{}
	for _, task := range tasks {
		result[task.ID] = task
		projects[task.ProjectID] = struct{}{}
	}
	for projectID := range projects {
		can, err := (&Project{ID: projectID}).CanWrite(s, a)
		if err != nil {
			return nil, err
		}
		if !can {
			return nil, ErrGenericForbidden{}
		}
	}
	return result, nil
}

func uniqueInt64s(values []int64) []int64 {
	seen := map[int64]struct{}{}
	out := make([]int64, 0, len(values))
	for _, value := range values {
		if value > 0 {
			if _, ok := seen[value]; !ok {
				seen[value] = struct{}{}
				out = append(out, value)
			}
		}
	}
	return out
}

func AddTasksToIteration(s *xorm.Session, a web.Auth, iterationID int64, taskIDs []int64, confirmTransfer bool) error {
	target, err := requireOwnedIteration(s, a, iterationID)
	if err != nil {
		return err
	}
	if target.Status == IterationStatusClosed {
		return ErrIterationClosed{IterationID: iterationID}
	}
	ids := uniqueInt64s(taskIDs)
	tasks, err := checkTaskWriteAccess(s, a, ids)
	if err != nil {
		return err
	}
	existing := []*IterationTask{}
	if err := s.In("task_id", ids).Find(&existing); err != nil {
		return err
	}
	byTask := map[int64]*IterationTask{}
	for _, assignment := range existing {
		byTask[assignment.TaskID] = assignment
	}
	for _, taskID := range ids {
		if current := byTask[taskID]; current != nil && current.IterationID != iterationID && !confirmTransfer {
			return ErrTaskAlreadyInIteration{TaskID: taskID, IterationID: current.IterationID}
		}
	}
	now := time.Now()
	for _, taskID := range ids {
		current := byTask[taskID]
		if current != nil && current.IterationID == iterationID {
			continue
		}
		sourceID := int64(0)
		if current != nil {
			sourceID = current.IterationID
			if err := closeHistory(s, taskID, current.IterationID, a.GetID(), IterationLeftTransferred, iterationID, now); err != nil {
				return err
			}
			if _, err := s.Where("task_id = ?", taskID).Delete(&IterationTask{}); err != nil {
				return err
			}
		}
		assignment := &IterationTask{IterationID: iterationID, TaskID: taskID, AddedAt: now, AddedByID: a.GetID(), SourceIterationID: sourceID}
		if _, err := s.Insert(assignment); err != nil {
			return err
		}
		task := tasks[taskID]
		history := &TaskIterationHistory{TaskID: taskID, IterationID: iterationID, JoinedAt: now, JoinedByID: a.GetID(), SourceIterationID: sourceID, TaskTitle: task.Title, ProjectID: task.ProjectID}
		if _, err := s.Insert(history); err != nil {
			return err
		}
	}
	return nil
}

func RemoveTasksFromIteration(s *xorm.Session, a web.Auth, iterationID int64, taskIDs []int64) error {
	iteration, err := requireOwnedIteration(s, a, iterationID)
	if err != nil {
		return err
	}
	if iteration.Status == IterationStatusClosed {
		return ErrIterationClosed{IterationID: iterationID}
	}
	ids := uniqueInt64s(taskIDs)
	if _, err := checkTaskWriteAccess(s, a, ids); err != nil {
		return err
	}
	now := time.Now()
	for _, taskID := range ids {
		has, err := s.Where("iteration_id = ? AND task_id = ?", iterationID, taskID).Exist(&IterationTask{})
		if err != nil {
			return err
		}
		if !has {
			continue
		}
		if err := closeHistory(s, taskID, iterationID, a.GetID(), IterationLeftRemoved, 0, now); err != nil {
			return err
		}
		if _, err := s.Where("iteration_id = ? AND task_id = ?", iterationID, taskID).Delete(&IterationTask{}); err != nil {
			return err
		}
	}
	return nil
}

func MoveTasksBetweenIterations(s *xorm.Session, a web.Auth, sourceID, targetID int64, taskIDs []int64) error {
	if _, err := requireOwnedIteration(s, a, sourceID); err != nil {
		return err
	}
	ids := uniqueInt64s(taskIDs)
	for _, taskID := range ids {
		has, err := s.Where("iteration_id = ? AND task_id = ?", sourceID, taskID).Exist(&IterationTask{})
		if err != nil {
			return err
		}
		if !has {
			return ErrTaskAlreadyInIteration{TaskID: taskID, IterationID: 0}
		}
	}
	return AddTasksToIteration(s, a, targetID, ids, true)
}

func closeHistory(s *xorm.Session, taskID, iterationID, leftBy int64, reason string, targetID int64, at time.Time) error {
	_, err := s.Where("task_id = ? AND iteration_id = ? AND left_at IS NULL", taskID, iterationID).
		Cols("left_at", "left_by_id", "left_reason", "target_iteration_id").
		Update(&TaskIterationHistory{LeftAt: &at, LeftByID: leftBy, LeftReason: reason, TargetIterationID: targetID})
	return err
}

func StartIteration(s *xorm.Session, a web.Auth, id int64) (*Iteration, error) {
	i, err := requireOwnedIteration(s, a, id)
	if err != nil {
		return nil, err
	}
	if i.Status != IterationStatusPlanned {
		return nil, ErrInvalidIterationTransition{IterationID: id}
	}
	if _, err := s.Where("id = ?", id).Cols("status").Update(&Iteration{Status: IterationStatusActive}); err != nil {
		return nil, err
	}
	i.Status = IterationStatusActive
	return i, hydrateIteration(s, i, a)
}

func ReopenIteration(s *xorm.Session, a web.Auth, id int64) (*Iteration, error) {
	i, err := requireOwnedIteration(s, a, id)
	if err != nil {
		return nil, err
	}
	if i.Status != IterationStatusClosed {
		return nil, ErrInvalidIterationTransition{IterationID: id}
	}
	if _, err := s.Exec("UPDATE iterations SET status = ?, closed_at = NULL, closed_by_id = 0 WHERE id = ?", IterationStatusActive, id); err != nil {
		return nil, err
	}
	i.Status, i.ClosedAt, i.ClosedByID = IterationStatusActive, nil, 0
	return i, hydrateIteration(s, i, a)
}

func CreateNextIteration(s *xorm.Session, a web.Auth, previousID int64, next *Iteration) (*Iteration, error) {
	previous, err := requireOwnedIteration(s, a, previousID)
	if err != nil {
		return nil, err
	}
	has, err := s.Where("previous_iteration_id = ?", previousID).Exist(&Iteration{})
	if err != nil {
		return nil, err
	}
	if has {
		return nil, ErrIterationAlreadyHasNext{IterationID: previousID}
	}
	duration := previous.EndDate.Sub(previous.StartDate)
	if next.StartDate.IsZero() {
		next.StartDate = previous.EndDate
	}
	if next.EndDate.IsZero() {
		next.EndDate = next.StartDate.Add(duration)
	}
	if next.Title == "" {
		next.Title = previous.Title + " · next"
	}
	next.PreviousIterationID = previousID
	if err := next.Create(s, a); err != nil {
		return nil, err
	}
	return next, nil
}

func CloseIteration(s *xorm.Session, a web.Auth, id int64, request *IterationCloseRequest) (*IterationCloseResult, error) {
	source, err := requireOwnedIteration(s, a, id)
	if err != nil {
		return nil, err
	}
	if source.Status == IterationStatusClosed {
		return nil, ErrIterationClosed{IterationID: id}
	}
	var target *Iteration
	if request.TargetIterationID > 0 {
		target, err = requireOwnedIteration(s, a, request.TargetIterationID)
		if err != nil {
			return nil, err
		}
		if target.Status == IterationStatusClosed || target.ID == source.ID {
			return nil, ErrInvalidIterationTransition{IterationID: target.ID}
		}
	}
	assignments := []*IterationTask{}
	if err := s.Where("iteration_id = ?", id).Find(&assignments); err != nil {
		return nil, err
	}
	allIDs := make([]int64, 0, len(assignments))
	for _, assignment := range assignments {
		allIDs = append(allIDs, assignment.TaskID)
	}
	tasks := map[int64]*Task{}
	if len(allIDs) > 0 {
		tasks, err = checkTaskWriteAccess(s, a, allIDs)
		if err != nil {
			return nil, err
		}
	}
	wanted := map[int64]struct{}{}
	for _, taskID := range uniqueInt64s(request.RolloverTaskIDs) {
		wanted[taskID] = struct{}{}
	}
	rolled, skipped := []int64{}, []int64{}
	now := time.Now()
	for _, assignment := range assignments {
		task := tasks[assignment.TaskID]
		_, shouldRoll := wanted[assignment.TaskID]
		if shouldRoll && task.Done {
			skipped = append(skipped, assignment.TaskID)
			shouldRoll = false
		}
		if shouldRoll && target == nil {
			return nil, ErrInvalidIterationTransition{IterationID: id}
		}
		reason, targetID := IterationLeftClosed, int64(0)
		if shouldRoll {
			reason, targetID = IterationLeftTransferred, target.ID
			rolled = append(rolled, assignment.TaskID)
		}
		if err := closeHistory(s, assignment.TaskID, id, a.GetID(), reason, targetID, now); err != nil {
			return nil, err
		}
	}
	if _, err := s.Where("iteration_id = ?", id).Delete(&IterationTask{}); err != nil {
		return nil, err
	}
	if len(rolled) > 0 {
		if err := AddTasksToIteration(s, a, target.ID, rolled, true); err != nil {
			return nil, err
		}
	}
	if _, err := s.Where("id = ?", id).Cols("status", "closed_at", "closed_by_id").Update(&Iteration{Status: IterationStatusClosed, ClosedAt: &now, ClosedByID: a.GetID()}); err != nil {
		return nil, err
	}
	source.Status, source.ClosedAt, source.ClosedByID = IterationStatusClosed, &now, a.GetID()
	if err := hydrateIteration(s, source, a); err != nil {
		return nil, err
	}
	if target != nil {
		if err := hydrateIteration(s, target, a); err != nil {
			return nil, err
		}
	}
	return &IterationCloseResult{Iteration: source, TargetIteration: target, RolledOverTaskIDs: rolled, SkippedCompletedTaskIDs: skipped}, nil
}

func GetTaskIterationInfo(s *xorm.Session, a web.Auth, taskID int64) (*TaskIterationInfo, error) {
	task := &Task{ID: taskID}
	can, _, err := task.CanRead(s, a)
	if err != nil {
		return nil, err
	}
	if !can {
		return nil, ErrGenericForbidden{}
	}
	info := &TaskIterationInfo{History: []*TaskIterationHistory{}}
	assignment := &IterationTask{}
	has, err := s.Where("task_id = ?", taskID).Get(assignment)
	if err != nil {
		return nil, err
	}
	if has {
		current, err := GetIterationByID(s, assignment.IterationID)
		if err != nil {
			return nil, err
		}
		if current.OwnerID == a.GetID() {
			if err := hydrateIteration(s, current, a); err != nil {
				return nil, err
			}
			info.Current = current
		}
	}
	err = s.Table("task_iteration_history").
		Select("task_iteration_history.*, iterations.title AS iteration_title, iterations.status AS iteration_status").
		Join("INNER", "iterations", "iterations.id = task_iteration_history.iteration_id AND iterations.owner_id = ?", a.GetID()).
		Where("task_iteration_history.task_id = ?", taskID).OrderBy("task_iteration_history.joined_at DESC").Find(&info.History)
	return info, err
}
