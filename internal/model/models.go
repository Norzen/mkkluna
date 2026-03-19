package model

import "time"

type User struct {
	ID           int64     `json:"id" db:"id"`
	Email        string    `json:"email" db:"email"`
	PasswordHash string    `json:"-" db:"password_hash"`
	Name         string    `json:"name" db:"name"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

type Team struct {
	ID          int64     `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	CreatedBy   int64     `json:"created_by" db:"created_by"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type TeamRole string

const (
	RoleOwner  TeamRole = "owner"
	RoleAdmin  TeamRole = "admin"
	RoleMember TeamRole = "member"
)

type TeamMember struct {
	ID       int64    `json:"id" db:"id"`
	TeamID   int64    `json:"team_id" db:"team_id"`
	UserID   int64    `json:"user_id" db:"user_id"`
	Role     TeamRole `json:"role" db:"role"`
	JoinedAt time.Time `json:"joined_at" db:"joined_at"`
}

type TaskStatus string

const (
	StatusTodo       TaskStatus = "todo"
	StatusInProgress TaskStatus = "in_progress"
	StatusDone       TaskStatus = "done"
)

type TaskPriority string

const (
	PriorityLow    TaskPriority = "low"
	PriorityMedium TaskPriority = "medium"
	PriorityHigh   TaskPriority = "high"
)

type Task struct {
	ID          int64        `json:"id" db:"id"`
	Title       string       `json:"title" db:"title"`
	Description string       `json:"description" db:"description"`
	Status      TaskStatus   `json:"status" db:"status"`
	Priority    TaskPriority `json:"priority" db:"priority"`
	TeamID      int64        `json:"team_id" db:"team_id"`
	AssigneeID  *int64       `json:"assignee_id" db:"assignee_id"`
	CreatedBy   int64        `json:"created_by" db:"created_by"`
	DueDate     *time.Time   `json:"due_date" db:"due_date"`
	CreatedAt   time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at" db:"updated_at"`
}

type TaskHistory struct {
	ID        int64     `json:"id" db:"id"`
	TaskID    int64     `json:"task_id" db:"task_id"`
	ChangedBy int64     `json:"changed_by" db:"changed_by"`
	FieldName string    `json:"field_name" db:"field_name"`
	OldValue  string    `json:"old_value" db:"old_value"`
	NewValue  string    `json:"new_value" db:"new_value"`
	ChangedAt time.Time `json:"changed_at" db:"changed_at"`
}

type TaskComment struct {
	ID        int64     `json:"id" db:"id"`
	TaskID    int64     `json:"task_id" db:"task_id"`
	UserID    int64     `json:"user_id" db:"user_id"`
	Content   string    `json:"content" db:"content"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// --- Request / Response DTOs ---

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type CreateTeamRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type InviteRequest struct {
	UserID int64    `json:"user_id"`
	Role   TeamRole `json:"role"`
}

type CreateTaskRequest struct {
	Title       string       `json:"title"`
	Description string       `json:"description"`
	Status      TaskStatus   `json:"status"`
	Priority    TaskPriority `json:"priority"`
	TeamID      int64        `json:"team_id"`
	AssigneeID  *int64       `json:"assignee_id"`
	DueDate     *time.Time   `json:"due_date"`
}

type UpdateTaskRequest struct {
	Title       *string       `json:"title"`
	Description *string       `json:"description"`
	Status      *TaskStatus   `json:"status"`
	Priority    *TaskPriority `json:"priority"`
	AssigneeID  *int64        `json:"assignee_id"`
	DueDate     *time.Time    `json:"due_date"`
}

type TaskFilter struct {
	TeamID     int64      `json:"team_id"`
	Status     TaskStatus `json:"status"`
	AssigneeID int64      `json:"assignee_id"`
	Page       int        `json:"page"`
	PageSize   int        `json:"page_size"`
}

type PaginatedResponse struct {
	Data       any `json:"data"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalCount int64       `json:"total_count"`
}

// --- Analytics DTOs ---

type TeamStats struct {
	TeamID       int64  `json:"team_id" db:"team_id"`
	TeamName     string `json:"team_name" db:"team_name"`
	MemberCount  int    `json:"member_count" db:"member_count"`
	DoneTasksWeek int   `json:"done_tasks_week" db:"done_tasks_week"`
}

type TopCreator struct {
	UserID    int64  `json:"user_id" db:"user_id"`
	UserName  string `json:"user_name" db:"user_name"`
	TeamID    int64  `json:"team_id" db:"team_id"`
	TeamName  string `json:"team_name" db:"team_name"`
	TaskCount int    `json:"task_count" db:"task_count"`
	Rank      int    `json:"rank" db:"rnk"`
}

type OrphanedTask struct {
	TaskID       int64  `json:"task_id" db:"task_id"`
	TaskTitle    string `json:"task_title" db:"task_title"`
	AssigneeID   int64  `json:"assignee_id" db:"assignee_id"`
	AssigneeName string `json:"assignee_name" db:"assignee_name"`
	TeamID       int64  `json:"team_id" db:"team_id"`
	TeamName     string `json:"team_name" db:"team_name"`
}
