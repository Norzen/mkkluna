package service

import (
	"cmp"
	"context"
	"database/sql"
	"errors"
	"fmt"

	"taskmanager/internal/cache"
	"taskmanager/internal/model"
	"taskmanager/internal/repository"
)

var ErrTaskNotFound = errors.New("task not found")

type TaskService interface {
	Create(ctx context.Context, userID int64, req model.CreateTaskRequest) (*model.Task, error)
	Update(ctx context.Context, userID, taskID int64, req model.UpdateTaskRequest) (*model.Task, error)
	List(ctx context.Context, filter model.TaskFilter) (*model.PaginatedResponse, error)
	GetHistory(ctx context.Context, taskID int64) ([]model.TaskHistory, error)
}

type taskService struct {
	taskRepo repository.TaskRepository
	teamRepo repository.TeamRepository
	cache    cache.Cache
}

func NewTaskService(taskRepo repository.TaskRepository, teamRepo repository.TeamRepository, c cache.Cache) TaskService {
	return &taskService{
		taskRepo: taskRepo,
		teamRepo: teamRepo,
		cache:    c,
	}
}

func (s *taskService) Create(ctx context.Context, userID int64, req model.CreateTaskRequest) (*model.Task, error) {
	if req.Title == "" {
		return nil, errors.New("task title is required")
	}
	if req.TeamID == 0 {
		return nil, errors.New("team_id is required")
	}

	isMember, err := s.teamRepo.IsMember(ctx, req.TeamID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, ErrNotMember
	}

	if req.AssigneeID != nil {
		assigneeMember, err := s.teamRepo.IsMember(ctx, req.TeamID, *req.AssigneeID)
		if err != nil {
			return nil, err
		}
		if !assigneeMember {
			return nil, errors.New("assignee is not a member of this team")
		}
	}

	status := cmp.Or(req.Status, model.StatusTodo)
	priority := cmp.Or(req.Priority, model.PriorityMedium)

	task := &model.Task{
		Title:       req.Title,
		Description: req.Description,
		Status:      status,
		Priority:    priority,
		TeamID:      req.TeamID,
		AssigneeID:  req.AssigneeID,
		CreatedBy:   userID,
		DueDate:     req.DueDate,
	}

	id, err := s.taskRepo.Create(ctx, task)
	if err != nil {
		return nil, err
	}
	task.ID = id

	_ = s.cache.DeleteByPattern(ctx, cache.TeamTasksPattern(req.TeamID))

	return task, nil
}

func (s *taskService) Update(ctx context.Context, userID, taskID int64, req model.UpdateTaskRequest) (*model.Task, error) {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}

	isMember, err := s.teamRepo.IsMember(ctx, task.TeamID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, ErrNotMember
	}

	var changes []model.TaskHistory

	if req.Title != nil && *req.Title != task.Title {
		changes = append(changes, model.TaskHistory{
			TaskID: taskID, ChangedBy: userID, FieldName: "title",
			OldValue: task.Title, NewValue: *req.Title,
		})
		task.Title = *req.Title
	}
	if req.Description != nil && *req.Description != task.Description {
		changes = append(changes, model.TaskHistory{
			TaskID: taskID, ChangedBy: userID, FieldName: "description",
			OldValue: task.Description, NewValue: *req.Description,
		})
		task.Description = *req.Description
	}
	if req.Status != nil && *req.Status != task.Status {
		changes = append(changes, model.TaskHistory{
			TaskID: taskID, ChangedBy: userID, FieldName: "status",
			OldValue: string(task.Status), NewValue: string(*req.Status),
		})
		task.Status = *req.Status
	}
	if req.Priority != nil && *req.Priority != task.Priority {
		changes = append(changes, model.TaskHistory{
			TaskID: taskID, ChangedBy: userID, FieldName: "priority",
			OldValue: string(task.Priority), NewValue: string(*req.Priority),
		})
		task.Priority = *req.Priority
	}
	if req.AssigneeID != nil {
		oldVal := "null"
		if task.AssigneeID != nil {
			oldVal = fmt.Sprintf("%d", *task.AssigneeID)
		}
		newVal := fmt.Sprintf("%d", *req.AssigneeID)
		if oldVal != newVal {
			assigneeMember, err := s.teamRepo.IsMember(ctx, task.TeamID, *req.AssigneeID)
			if err != nil {
				return nil, err
			}
			if !assigneeMember {
				return nil, errors.New("assignee is not a member of this team")
			}
			changes = append(changes, model.TaskHistory{
				TaskID: taskID, ChangedBy: userID, FieldName: "assignee_id",
				OldValue: oldVal, NewValue: newVal,
			})
			task.AssigneeID = req.AssigneeID
		}
	}

	if err := s.taskRepo.Update(ctx, task); err != nil {
		return nil, err
	}

	for i := range changes {
		_ = s.taskRepo.AddHistory(ctx, &changes[i])
	}

	_ = s.cache.DeleteByPattern(ctx, cache.TeamTasksPattern(task.TeamID))
	_ = s.cache.Delete(ctx, cache.TaskKey(taskID))

	return task, nil
}

func (s *taskService) List(ctx context.Context, filter model.TaskFilter) (*model.PaginatedResponse, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 || filter.PageSize > 100 {
		filter.PageSize = 20
	}

	cacheKey := cache.TaskListKey(filter.TeamID, string(filter.Status), filter.Page, filter.PageSize)

	var cached model.PaginatedResponse
	if err := s.cache.Get(ctx, cacheKey, &cached); err == nil {
		return &cached, nil
	}

	tasks, total, err := s.taskRepo.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	resp := &model.PaginatedResponse{
		Data:       tasks,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalCount: total,
	}

	_ = s.cache.Set(ctx, cacheKey, resp, 0)

	return resp, nil
}

func (s *taskService) GetHistory(ctx context.Context, taskID int64) ([]model.TaskHistory, error) {
	return s.taskRepo.GetHistory(ctx, taskID)
}
