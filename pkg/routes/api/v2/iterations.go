// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.

package apiv2

import (
	"context"
	"fmt"
	"net/http"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/events"
	"code.vikunja.io/api/pkg/models"
	"code.vikunja.io/api/pkg/web"
	"code.vikunja.io/api/pkg/web/handler"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/conditional"
	"xorm.io/xorm"
)

type iterationListBody struct{ Body Paginated[*models.Iteration] }
type iterationTaskListBody struct{ Body Paginated[*models.Task] }
type iterationReadBody struct {
	models.Iteration
	MaxPermission models.Permission `json:"max_permission" readOnly:"true"`
}

func RegisterIterationRoutes(api huma.API) {
	tags := []string{"iterations"}
	Register(api, huma.Operation{OperationID: "iterations-list", Summary: "List iterations", Method: http.MethodGet, Path: "/iterations", Tags: tags}, iterationsList)
	Register(api, huma.Operation{OperationID: "iterations-create", Summary: "Create an iteration", Method: http.MethodPost, Path: "/iterations", Tags: tags}, iterationsCreate)
	Register(api, huma.Operation{OperationID: "iterations-read", Summary: "Get an iteration", Method: http.MethodGet, Path: "/iterations/{iteration}", Tags: tags}, iterationsRead)
	Register(api, huma.Operation{OperationID: "iterations-update", Summary: "Update an iteration", Method: http.MethodPut, Path: "/iterations/{iteration}", Tags: tags}, iterationsUpdate)
	Register(api, huma.Operation{OperationID: "iterations-delete", Summary: "Delete an empty planned iteration", Method: http.MethodDelete, Path: "/iterations/{iteration}", Tags: tags}, iterationsDelete)
	Register(api, huma.Operation{OperationID: "iteration-tasks-list", Summary: "List tasks in an iteration", Method: http.MethodGet, Path: "/iterations/{iteration}/tasks", Tags: tags}, iterationTasksList)
	Register(api, huma.Operation{OperationID: "iteration-tasks-add", Summary: "Add tasks to an iteration", Method: http.MethodPost, Path: "/iterations/{iteration}/tasks/bulk", Tags: tags}, iterationTasksAdd)
	Register(api, huma.Operation{OperationID: "iteration-tasks-remove", Summary: "Remove tasks from an iteration", Method: http.MethodDelete, Path: "/iterations/{iteration}/tasks/bulk", Tags: tags}, iterationTasksRemove)
	Register(api, huma.Operation{OperationID: "iteration-tasks-move", Summary: "Move iteration tasks", Method: http.MethodPost, Path: "/iterations/{iteration}/tasks/move", DefaultStatus: http.StatusOK, Tags: tags}, iterationTasksMove)
	Register(api, huma.Operation{OperationID: "iteration-start", Summary: "Start an iteration", Method: http.MethodPost, Path: "/iterations/{iteration}/start", DefaultStatus: http.StatusOK, Tags: tags}, iterationStart)
	Register(api, huma.Operation{OperationID: "iteration-reopen", Summary: "Reopen an iteration", Method: http.MethodPost, Path: "/iterations/{iteration}/reopen", DefaultStatus: http.StatusOK, Tags: tags}, iterationReopen)
	Register(api, huma.Operation{OperationID: "iteration-next", Summary: "Create the next iteration", Method: http.MethodPost, Path: "/iterations/{iteration}/next", Tags: tags}, iterationNext)
	Register(api, huma.Operation{OperationID: "iteration-close", Summary: "Close an iteration and roll tasks over", Method: http.MethodPost, Path: "/iterations/{iteration}/close", DefaultStatus: http.StatusOK, Tags: tags}, iterationClose)
	Register(api, huma.Operation{OperationID: "task-iterations-read", Summary: "Get a task's iteration history", Method: http.MethodGet, Path: "/tasks/{task}/iterations", Tags: tags}, taskIterationsRead)
	Register(api, huma.Operation{OperationID: "task-iteration-set", Summary: "Set a task's current iteration", Method: http.MethodPut, Path: "/tasks/{task}/iteration", Tags: tags}, taskIterationSet)
}

func init() { AddRouteRegistrar(RegisterIterationRoutes) }

func iterationsList(ctx context.Context, in *struct {
	ListParams
	Status  models.IterationStatus `query:"status" enum:"planned,active,closed"`
	SortBy  string                 `query:"sort_by" enum:"start_date,end_date,title,created" default:"start_date"`
	OrderBy string                 `query:"order_by" enum:"asc,desc" default:"desc"`
	Format  string                 `query:"format" enum:"html,markdown"`
}) (*iterationListBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	m := &models.Iteration{FilterStatus: in.Status, SortBy: in.SortBy, OrderBy: in.OrderBy}
	result, _, total, err := handler.DoReadAll(ctx, m, a, in.Q, in.Page, in.PerPage)
	if err != nil {
		return nil, translateDomainError(err)
	}
	items, ok := result.([]*models.Iteration)
	if !ok {
		return nil, fmt.Errorf("iteration ReadAll returned %T", result)
	}
	for _, item := range items {
		convertToMarkdown(ctx, &item.Description)
	}
	return &iterationListBody{Body: NewPaginated(items, total, in.Page, in.PerPage)}, nil
}

func iterationsCreate(ctx context.Context, in *struct {
	Format string `query:"format" enum:"html,markdown"`
	Body   models.Iteration
}) (*singleBody[models.Iteration], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if err := convertToHTML(ctx, &in.Body.Description); err != nil {
		return nil, translateDomainError(err)
	}
	if err := handler.DoCreate(ctx, &in.Body, a); err != nil {
		return nil, translateDomainError(err)
	}
	convertToMarkdown(ctx, &in.Body.Description)
	return &singleBody[models.Iteration]{Body: &in.Body}, nil
}

func iterationsRead(ctx context.Context, in *struct {
	ID     int64  `path:"iteration"`
	Format string `query:"format" enum:"html,markdown"`
	conditional.Params
}) (*singleReadBody[iterationReadBody], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	i := &models.Iteration{ID: in.ID}
	maxPermission, err := handler.DoReadOne(ctx, i, a)
	if err != nil {
		return nil, translateDomainError(err)
	}
	body := &iterationReadBody{Iteration: *i, MaxPermission: models.Permission(maxPermission)}
	convertToMarkdown(ctx, &body.Description)
	return conditionalReadResponse(&in.Params, body, i.Updated, maxPermission)
}

func iterationsUpdate(ctx context.Context, in *struct {
	ID     int64  `path:"iteration"`
	Format string `query:"format" enum:"html,markdown"`
	Body   iterationReadBody
}) (*singleBody[models.Iteration], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	i := &in.Body.Iteration
	i.ID = in.ID
	if err := convertToHTML(ctx, &i.Description); err != nil {
		return nil, translateDomainError(err)
	}
	if err := handler.DoUpdate(ctx, i, a); err != nil {
		return nil, translateDomainError(err)
	}
	convertToMarkdown(ctx, &i.Description)
	return &singleBody[models.Iteration]{Body: i}, nil
}

func iterationsDelete(ctx context.Context, in *struct {
	ID int64 `path:"iteration"`
}) (*emptyBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if err := handler.DoDelete(ctx, &models.Iteration{ID: in.ID}, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &emptyBody{}, nil
}

func iterationTasksList(ctx context.Context, in *struct {
	ListParams
	ID        int64  `path:"iteration"`
	ProjectID string `query:"project_id"`
	Status    string `query:"status" enum:"open,done"`
	SortBy    string `query:"sort_by" enum:"title,done,priority,due_date,start_date,id" default:"done"`
	OrderBy   string `query:"order_by" enum:"asc,desc" default:"asc"`
	Format    string `query:"format" enum:"html,markdown"`
}) (*iterationTaskListBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	s := db.NewSession()
	defer s.Close()
	tasks, total, err := models.ListIterationTasks(s, a, in.ID, in.Q, in.ProjectID, in.Status, in.SortBy, in.OrderBy, in.Page, in.PerPage)
	if err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}
	if err := s.Commit(); err != nil {
		return nil, translateDomainError(err)
	}
	convertTasksToMarkdown(ctx, tasks...)
	return &iterationTaskListBody{Body: NewPaginated(tasks, total, in.Page, in.PerPage)}, nil
}

func runIterationAction[T any](ctx context.Context, action func(*xorm.Session, web.Auth) (*T, error)) (*singleBody[T], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	s := db.NewSession()
	defer s.Close()
	result, err := action(s, a)
	if err != nil {
		_ = s.Rollback()
		events.CleanupPending(s)
		return nil, translateDomainError(err)
	}
	if err := s.Commit(); err != nil {
		events.CleanupPending(s)
		return nil, translateDomainError(err)
	}
	events.DispatchPending(ctx, s)
	return &singleBody[T]{Body: result}, nil
}

func iterationTasksAdd(ctx context.Context, in *struct {
	ID   int64 `path:"iteration"`
	Body models.IterationTaskChange
}) (*singleBody[models.Iteration], error) {
	return runIterationAction(ctx, func(s *xorm.Session, a web.Auth) (*models.Iteration, error) {
		if err := models.AddTasksToIteration(s, a, in.ID, in.Body.TaskIDs, in.Body.ConfirmTransfer); err != nil {
			return nil, err
		}
		i, err := models.GetIterationByID(s, in.ID)
		if err != nil {
			return nil, err
		}
		return i, nil
	})
}

func iterationTasksRemove(ctx context.Context, in *struct {
	ID   int64 `path:"iteration"`
	Body models.IterationTaskChange
}) (*emptyBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	s := db.NewSession()
	defer s.Close()
	if err := models.RemoveTasksFromIteration(s, a, in.ID, in.Body.TaskIDs); err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}
	if err := s.Commit(); err != nil {
		return nil, translateDomainError(err)
	}
	return &emptyBody{}, nil
}

func iterationTasksMove(ctx context.Context, in *struct {
	ID   int64 `path:"iteration"`
	Body models.IterationMoveTasks
}) (*singleBody[models.Iteration], error) {
	return runIterationAction(ctx, func(s *xorm.Session, a web.Auth) (*models.Iteration, error) {
		if err := models.MoveTasksBetweenIterations(s, a, in.ID, in.Body.TargetIterationID, in.Body.TaskIDs); err != nil {
			return nil, err
		}
		return models.GetIterationByID(s, in.Body.TargetIterationID)
	})
}

func iterationStart(ctx context.Context, in *struct {
	ID int64 `path:"iteration"`
}) (*singleBody[models.Iteration], error) {
	return runIterationAction(ctx, func(s *xorm.Session, a web.Auth) (*models.Iteration, error) {
		return models.StartIteration(s, a, in.ID)
	})
}
func iterationReopen(ctx context.Context, in *struct {
	ID int64 `path:"iteration"`
}) (*singleBody[models.Iteration], error) {
	return runIterationAction(ctx, func(s *xorm.Session, a web.Auth) (*models.Iteration, error) {
		return models.ReopenIteration(s, a, in.ID)
	})
}
func iterationNext(ctx context.Context, in *struct {
	ID     int64  `path:"iteration"`
	Format string `query:"format" enum:"html,markdown"`
	Body   models.Iteration
}) (*singleBody[models.Iteration], error) {
	if err := convertToHTML(ctx, &in.Body.Description); err != nil {
		return nil, translateDomainError(err)
	}
	response, err := runIterationAction(ctx, func(s *xorm.Session, a web.Auth) (*models.Iteration, error) {
		return models.CreateNextIteration(s, a, in.ID, &in.Body)
	})
	if response != nil {
		convertToMarkdown(ctx, &response.Body.Description)
	}
	return response, err
}
func iterationClose(ctx context.Context, in *struct {
	ID   int64 `path:"iteration"`
	Body models.IterationCloseRequest
}) (*singleBody[models.IterationCloseResult], error) {
	return runIterationAction(ctx, func(s *xorm.Session, a web.Auth) (*models.IterationCloseResult, error) {
		return models.CloseIteration(s, a, in.ID, &in.Body)
	})
}

func taskIterationsRead(ctx context.Context, in *struct {
	TaskID int64 `path:"task"`
}) (*singleBody[models.TaskIterationInfo], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	s := db.NewSession()
	defer s.Close()
	info, err := models.GetTaskIterationInfo(s, a, in.TaskID)
	if err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}
	if err := s.Commit(); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[models.TaskIterationInfo]{Body: info}, nil
}

func taskIterationSet(ctx context.Context, in *struct {
	TaskID int64 `path:"task"`
	Body   struct {
		IterationID     int64 `json:"iteration_id"`
		ConfirmTransfer bool  `json:"confirm_transfer"`
	}
}) (*singleBody[models.TaskIterationInfo], error) {
	return runIterationAction(ctx, func(s *xorm.Session, a web.Auth) (*models.TaskIterationInfo, error) {
		current := &models.IterationTask{}
		has, err := s.Where("task_id = ?", in.TaskID).Get(current)
		if err != nil {
			return nil, err
		}
		if in.Body.IterationID == 0 {
			if has {
				if err := models.RemoveTasksFromIteration(s, a, current.IterationID, []int64{in.TaskID}); err != nil {
					return nil, err
				}
			}
		} else if err := models.AddTasksToIteration(s, a, in.Body.IterationID, []int64{in.TaskID}, in.Body.ConfirmTransfer); err != nil {
			return nil, err
		}
		return models.GetTaskIterationInfo(s, a, in.TaskID)
	})
}
