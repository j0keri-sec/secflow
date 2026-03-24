package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/secflow/server/internal/model"
)

// ErrNotFound is returned when a document is not found.
var ErrNotFound = errors.New("not found")

// VulnRepo handles all vuln_records operations.
type VulnRepo struct{ db *DB }

func NewVulnRepo(db *DB) *VulnRepo { return &VulnRepo{db: db} }

// UpsertResult describes what changed during an upsert.
type UpsertResult struct {
	IsNew           bool
	SeverityChanged bool
	TagsChanged     bool
	OldSeverity     string
	OldTags         []string
}

// NeedsRenotify returns true when the change warrants a new push notification.
func (r *UpsertResult) NeedsRenotify() bool {
	return r.IsNew || r.SeverityChanged || r.TagsChanged
}

// Upsert inserts or updates a vuln record, returning what changed.
func (r *VulnRepo) Upsert(ctx context.Context, v *model.VulnRecord) (*UpsertResult, error) {
	coll := r.db.coll(model.CollVulnRecords)
	filter := bson.M{"key": v.Key}

	log.Debug().Str("key", v.Key).Str("title", v.Title).Msg("Upsert vuln start")

	var existing model.VulnRecord
	err := coll.FindOne(ctx, filter).Decode(&existing)

	result := &UpsertResult{}
	now := nowUTC()

	if errors.Is(err, mongo.ErrNoDocuments) {
		// New document.
		result.IsNew = true
		v.CreatedAt = now
		v.UpdatedAt = now
		log.Debug().Str("key", v.Key).Msg("Inserting new vuln")
		if _, err = coll.InsertOne(ctx, v); err != nil {
			log.Error().Err(err).Str("key", v.Key).Msg("Failed to insert vuln")
			return nil, err
		}
		log.Debug().Str("key", v.Key).Msg("Inserted vuln successfully")
		return result, nil
	}
	if err != nil {
		log.Error().Err(err).Str("key", v.Key).Msg("Failed to find vuln")
		return nil, err
	}

	// Existing document — check for meaningful changes.
	if string(existing.Severity) != string(v.Severity) {
		result.SeverityChanged = true
		result.OldSeverity = string(existing.Severity)
	}
	if !tagsSubset(v.Tags, existing.Tags) {
		result.TagsChanged = true
		result.OldTags = existing.Tags
		v.Tags = mergeUnique(existing.Tags, v.Tags)
	}

	update := bson.M{
		"$set": bson.M{
			"title":       v.Title,
			"description": v.Description,
			"severity":    v.Severity,
			"solutions":   v.Solutions,
			"references":  v.References,
			"tags":        v.Tags,
			"updated_at":  now,
		},
	}
	_, err = coll.UpdateOne(ctx, filter, update)
	return result, err
}

// MarkPushed sets pushed=true for the given key.
func (r *VulnRepo) MarkPushed(ctx context.Context, key string) error {
	_, err := r.db.coll(model.CollVulnRecords).UpdateOne(ctx,
		bson.M{"key": key},
		bson.M{"$set": bson.M{"pushed": true, "updated_at": nowUTC()}},
	)
	return err
}

// GetByKey returns a single vuln record by its unique key or MongoDB ObjectID.
// It first tries to parse the key as an ObjectID, then falls back to key lookup.
func (r *VulnRepo) GetByKey(ctx context.Context, key string) (*model.VulnRecord, error) {
	var v model.VulnRecord
	
	// First try to find by _id (ObjectID)
	if oid, err := bson.ObjectIDFromHex(key); err == nil {
		err := r.db.coll(model.CollVulnRecords).FindOne(ctx, bson.M{"_id": oid}).Decode(&v)
		if err == nil {
			return &v, nil
		}
		if !errors.Is(err, mongo.ErrNoDocuments) {
			return nil, err
		}
	}
	
	// Fall back to key lookup
	err := r.db.coll(model.CollVulnRecords).FindOne(ctx, bson.M{"key": key}).Decode(&v)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	return &v, err
}

// FindPushedByCVE returns all pushed records sharing the given CVE ID.
func (r *VulnRepo) FindPushedByCVE(ctx context.Context, cve string) ([]*model.VulnRecord, error) {
	cursor, err := r.db.coll(model.CollVulnRecords).Find(ctx, bson.M{"cve": cve, "pushed": true})
	if err != nil {
		return nil, err
	}
	var records []*model.VulnRecord
	return records, cursor.All(ctx, &records)
}

// UpdateGithubSearch updates the github_search field for a given key.
func (r *VulnRepo) UpdateGithubSearch(ctx context.Context, key string, links []string) error {
	_, err := r.db.coll(model.CollVulnRecords).UpdateOne(ctx,
		bson.M{"key": key},
		bson.M{"$set": bson.M{"github_search": links, "updated_at": nowUTC()}},
	)
	return err
}

// Count returns the total number of vuln records.
func (r *VulnRepo) Count(ctx context.Context) (int64, error) {
	return r.db.coll(model.CollVulnRecords).CountDocuments(ctx, bson.M{})
}

// ListFilter defines parameters for paginated listing.
type VulnListFilter struct {
	Severity string
	Source   string
	CVE      string
	Pushed   *bool
	Keywords string
	Page     int64
	PageSize int64
}

// List returns paginated vuln records.
func (r *VulnRepo) List(ctx context.Context, f VulnListFilter) ([]*model.VulnRecord, int64, error) {
	query := bson.M{}
	if f.Severity != "" {
		query["severity"] = f.Severity
	}
	if f.Source != "" {
		query["source"] = f.Source
	}
	if f.CVE != "" {
		query["cve"] = bson.M{"$regex": f.CVE, "$options": "i"}
	}
	if f.Pushed != nil {
		query["pushed"] = *f.Pushed
	}
	if f.Keywords != "" {
		query["$or"] = bson.A{
			bson.M{"title": bson.M{"$regex": f.Keywords, "$options": "i"}},
			bson.M{"description": bson.M{"$regex": f.Keywords, "$options": "i"}},
		}
	}

	coll := r.db.coll(model.CollVulnRecords)
	total, err := coll.CountDocuments(ctx, query)
	if err != nil {
		return nil, 0, err
	}

	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 {
		f.PageSize = 20
	}
	skip := (f.Page - 1) * f.PageSize

	cursor, err := coll.Find(ctx, query,
		options.Find().
			SetSort(bson.D{{Key: "created_at", Value: -1}}).
			SetSkip(skip).
			SetLimit(f.PageSize),
	)
	if err != nil {
		return nil, 0, err
	}
	var records []*model.VulnRecord
	return records, total, cursor.All(ctx, &records)
}

// Delete removes a vuln record by ID.
func (r *VulnRepo) Delete(ctx context.Context, id bson.ObjectID) error {
	_, err := r.db.coll(model.CollVulnRecords).DeleteOne(ctx, bson.M{"_id": id})
	return err
}

// --------------------------------------------------------------------------
// helpers
// --------------------------------------------------------------------------

// tagsSubset returns true when all newTags are already present in existing.
func tagsSubset(newTags, existing []string) bool {
	set := make(map[string]struct{}, len(existing))
	for _, t := range existing {
		set[t] = struct{}{}
	}
	for _, t := range newTags {
		if _, ok := set[t]; !ok {
			return false
		}
	}
	return true
}

// mergeUnique merges two slices and removes duplicates.
func mergeUnique(a, b []string) []string {
	seen := make(map[string]struct{}, len(a)+len(b))
	for _, s := range a {
		seen[s] = struct{}{}
	}
	for _, s := range b {
		seen[s] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	return out
}

// --------------------------------------------------------------------------
// UserRepo
// --------------------------------------------------------------------------

// UserRepo handles all users collection operations.
type UserRepo struct{ db *DB }

func NewUserRepo(db *DB) *UserRepo { return &UserRepo{db: db} }

// DB exposes the underlying *DB for cases where no typed method exists.
func (r *UserRepo) DB() *DB { return r.db }

// CountAll returns the total number of users in the database.
func (r *UserRepo) CountAll(ctx context.Context) (int64, error) {
	return r.db.coll(model.CollUsers).CountDocuments(ctx, bson.M{})
}

// Create inserts a new user document.
func (r *UserRepo) Create(ctx context.Context, u *model.User) error {
	u.CreatedAt = nowUTC()
	u.UpdatedAt = nowUTC()
	_, err := r.db.coll(model.CollUsers).InsertOne(ctx, u)
	return err
}

// GetByUsername returns a user by username, or nil.
func (r *UserRepo) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	var u model.User
	err := r.db.coll(model.CollUsers).FindOne(ctx, bson.M{"username": username}).Decode(&u)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	return &u, err
}

// GetByEmail returns a user by email address (case-insensitive).
func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	var u model.User
	// Email lookup is case-insensitive using regex
	err := r.db.coll(model.CollUsers).FindOne(ctx, bson.M{"email": bson.M{"$regex": "^" + email + "$", "$options": "i"}}).Decode(&u)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	return &u, err
}

// GetByID returns a user by ObjectID.
func (r *UserRepo) GetByID(ctx context.Context, id bson.ObjectID) (*model.User, error) {
	var u model.User
	err := r.db.coll(model.CollUsers).FindOne(ctx, bson.M{"_id": id}).Decode(&u)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	return &u, err
}

// List returns all users (admin only).
func (r *UserRepo) List(ctx context.Context, page, pageSize int64) ([]*model.User, int64, error) {
	coll := r.db.coll(model.CollUsers)
	total, _ := coll.CountDocuments(ctx, bson.M{})
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	cursor, err := coll.Find(ctx, bson.M{},
		options.Find().
			SetSort(bson.D{{Key: "created_at", Value: -1}}).
			SetSkip((page-1)*pageSize).
			SetLimit(pageSize),
	)
	if err != nil {
		return nil, 0, err
	}
	var users []*model.User
	return users, total, cursor.All(ctx, &users)
}

// UpdatePassword changes a user's password hash.
func (r *UserRepo) UpdatePassword(ctx context.Context, id bson.ObjectID, hash string) error {
	_, err := r.db.coll(model.CollUsers).UpdateOne(ctx,
		bson.M{"_id": id},
		bson.M{"$set": bson.M{"password_hash": hash, "updated_at": nowUTC()}},
	)
	return err
}

// UpdateRole changes a user's role.
func (r *UserRepo) UpdateRole(ctx context.Context, id bson.ObjectID, role model.RoleType) error {
	_, err := r.db.coll(model.CollUsers).UpdateOne(ctx,
		bson.M{"_id": id},
		bson.M{"$set": bson.M{"role": role, "updated_at": nowUTC()}},
	)
	return err
}

// Delete removes a user.
func (r *UserRepo) Delete(ctx context.Context, id bson.ObjectID) error {
	_, err := r.db.coll(model.CollUsers).DeleteOne(ctx, bson.M{"_id": id})
	return err
}

// --------------------------------------------------------------------------
// InviteCodeRepo
// --------------------------------------------------------------------------

// InviteCodeRepo handles invite_codes operations.
type InviteCodeRepo struct{ db *DB }

func NewInviteCodeRepo(db *DB) *InviteCodeRepo { return &InviteCodeRepo{db: db} }

// Create inserts a new invite code.
func (r *InviteCodeRepo) Create(ctx context.Context, c *model.InviteCode) error {
	c.CreatedAt = nowUTC()
	_, err := r.db.coll(model.CollInviteCodes).InsertOne(ctx, c)
	return err
}

// GetByCode returns an invite code document.
func (r *InviteCodeRepo) GetByCode(ctx context.Context, code string) (*model.InviteCode, error) {
	var c model.InviteCode
	err := r.db.coll(model.CollInviteCodes).FindOne(ctx, bson.M{"code": code}).Decode(&c)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	return &c, err
}

// CountByOwner counts how many codes a given user has generated.
func (r *InviteCodeRepo) CountByOwner(ctx context.Context, ownerID bson.ObjectID) (int64, error) {
	return r.db.coll(model.CollInviteCodes).CountDocuments(ctx, bson.M{"owner_id": ownerID})
}

// MarkUsed marks a code as used by a specific user.
func (r *InviteCodeRepo) MarkUsed(ctx context.Context, code string, usedBy bson.ObjectID) error {
	_, err := r.db.coll(model.CollInviteCodes).UpdateOne(ctx,
		bson.M{"code": code},
		bson.M{"$set": bson.M{
			"used":       true,
			"used_by_id": usedBy,
			"used_at":    nowUTC(),
		}},
	)
	return err
}

// ListByOwner returns all invite codes belonging to the specified user.
func (r *InviteCodeRepo) ListByOwner(ctx context.Context, ownerID bson.ObjectID) ([]*model.InviteCode, error) {
	cursor, err := r.db.coll(model.CollInviteCodes).Find(ctx,
		bson.M{"owner_id": ownerID},
		options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}),
	)
	if err != nil {
		return nil, err
	}
	var codes []*model.InviteCode
	return codes, cursor.All(ctx, &codes)
}

// --------------------------------------------------------------------------
// NodeRepo
// --------------------------------------------------------------------------

// NodeRepo handles the nodes collection.
type NodeRepo struct{ db *DB }

func NewNodeRepo(db *DB) *NodeRepo { return &NodeRepo{db: db} }

// Upsert inserts or updates a node document by node_id.
func (r *NodeRepo) Upsert(ctx context.Context, n *model.Node) error {
	n.LastSeenAt = nowUTC()
	_, err := r.db.coll(model.CollNodes).UpdateOne(ctx,
		bson.M{"node_id": n.NodeID},
		bson.M{
			"$set": bson.M{
				"name":         n.Name,
				"status":       n.Status,
				"info":         n.Info,
				"sources":      n.Sources,
				"last_seen_at": n.LastSeenAt,
			},
			"$setOnInsert": bson.M{
				"token":         n.Token,
				"registered_at": nowUTC(),
			},
		},
		options.UpdateOne().SetUpsert(true),
	)
	return err
}

// SetStatus updates only the status of a node.
func (r *NodeRepo) SetStatus(ctx context.Context, nodeID string, status model.NodeStatus) error {
	_, err := r.db.coll(model.CollNodes).UpdateOne(ctx,
		bson.M{"node_id": nodeID},
		bson.M{"$set": bson.M{"status": status, "last_seen_at": nowUTC()}},
	)
	return err
}

// UpdateInfo updates the node info fields from heartbeat data.
func (r *NodeRepo) UpdateInfo(ctx context.Context, nodeID string, info model.NodeInfo) error {
	_, err := r.db.coll(model.CollNodes).UpdateOne(ctx,
		bson.M{"node_id": nodeID},
		bson.M{"$set": bson.M{
			"info":         info,
			"last_seen_at": nowUTC(),
		}},
	)
	return err
}

// UpdateTaskStats updates the node task performance statistics.
func (r *NodeRepo) UpdateTaskStats(ctx context.Context, nodeID string, stats model.NodeTaskStats) error {
	_, err := r.db.coll(model.CollNodes).UpdateOne(ctx,
		bson.M{"node_id": nodeID},
		bson.M{"$set": bson.M{
			"task_stats": stats,
		}},
	)
	return err
}

// GetByNodeID retrieves a node by its stable UUID.
func (r *NodeRepo) GetByNodeID(ctx context.Context, nodeID string) (*model.Node, error) {
	var n model.Node
	err := r.db.coll(model.CollNodes).FindOne(ctx, bson.M{"node_id": nodeID}).Decode(&n)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	return &n, err
}

// GetByID retrieves a node by its MongoDB ObjectID.
func (r *NodeRepo) GetByID(ctx context.Context, id bson.ObjectID) (*model.Node, error) {
	var n model.Node
	err := r.db.coll(model.CollNodes).FindOne(ctx, bson.M{"_id": id}).Decode(&n)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	return &n, err
}

// Delete removes a node by its MongoDB ObjectID.
func (r *NodeRepo) Delete(ctx context.Context, id bson.ObjectID) error {
	_, err := r.db.coll(model.CollNodes).DeleteOne(ctx, bson.M{"_id": id})
	return err
}

// UpdateToken updates the token for a node.
func (r *NodeRepo) UpdateToken(ctx context.Context, id bson.ObjectID, token string) error {
	_, err := r.db.coll(model.CollNodes).UpdateOne(ctx,
		bson.M{"_id": id},
		bson.M{"$set": bson.M{"token": token, "updated_at": nowUTC()}},
	)
	return err
}

// List returns all nodes.
func (r *NodeRepo) List(ctx context.Context) ([]*model.Node, error) {
	cursor, err := r.db.coll(model.CollNodes).Find(ctx, bson.M{},
		options.Find().SetSort(bson.D{{Key: "last_seen_at", Value: -1}}),
	)
	if err != nil {
		return nil, err
	}
	var nodes []*model.Node
	return nodes, cursor.All(ctx, &nodes)
}

// --------------------------------------------------------------------------
// TaskRepo
// --------------------------------------------------------------------------

// TaskRepo handles tasks collection.
type TaskRepo struct{ db *DB }

func NewTaskRepo(db *DB) *TaskRepo { return &TaskRepo{db: db} }

// Create inserts a new task.
func (r *TaskRepo) Create(ctx context.Context, t *model.Task) error {
	t.CreatedAt = nowUTC()
	t.UpdatedAt = nowUTC()
	_, err := r.db.coll(model.CollTasks).InsertOne(ctx, t)
	return err
}

// GetByTaskID retrieves a task by its string UUID.
func (r *TaskRepo) GetByTaskID(ctx context.Context, taskID string) (*model.Task, error) {
	var t model.Task
	err := r.db.coll(model.CollTasks).FindOne(ctx, bson.M{"task_id": taskID}).Decode(&t)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	return &t, err
}

// UpdateStatus changes the task status and optionally sets assigned_to.
func (r *TaskRepo) UpdateStatus(ctx context.Context, taskID string, status model.TaskStatus, assignedTo string) error {
	set := bson.M{"status": status, "updated_at": nowUTC()}
	if assignedTo != "" {
		set["assigned_to"] = assignedTo
	}
	if status == model.TaskDone || status == model.TaskFailed {
		set["finished_at"] = nowUTC()
	}
	_, err := r.db.coll(model.CollTasks).UpdateOne(ctx,
		bson.M{"task_id": taskID},
		bson.M{"$set": set},
	)
	return err
}

// UpdateProgress updates the progress percentage of a task (0–100).
func (r *TaskRepo) UpdateProgress(ctx context.Context, taskID string, progress int) error {
	_, err := r.db.coll(model.CollTasks).UpdateOne(ctx,
		bson.M{"task_id": taskID},
		bson.M{"$set": bson.M{"progress": progress, "updated_at": nowUTC()}},
	)
	return err
}

// SetResult saves the result payload returned by the client.
func (r *TaskRepo) SetResult(ctx context.Context, taskID string, result []byte, errMsg string) error {
	set := bson.M{
		"result":      result,
		"status":      model.TaskDone,
		"progress":    100,
		"updated_at":  nowUTC(),
		"finished_at": nowUTC(),
	}
	if errMsg != "" {
		set["error"] = errMsg
		set["status"] = model.TaskFailed
	}
	_, err := r.db.coll(model.CollTasks).UpdateOne(ctx,
		bson.M{"task_id": taskID},
		bson.M{"$set": set},
	)
	return err
}

// UpdateRetryMetadata updates the retry tracking fields for a task.
func (r *TaskRepo) UpdateRetryMetadata(ctx context.Context, taskID string, retryCount int, maxRetries int, errorMsg string) error {
	set := bson.M{
		"retry_count":  retryCount,
		"max_retries":  maxRetries,
		"last_retry_at": nowUTC(),
		"updated_at":    nowUTC(),
	}
	
	if errorMsg != "" {
		// Add error to retry_errors array (keep last 5 errors)
		push := bson.M{
			"retry_errors": bson.M{
				"$each":     []string{fmt.Sprintf("%s: %s", nowUTC().Format(time.RFC3339), errorMsg)},
				"$slice":    -5, // Keep last 5 errors
				"$position": 0,
			},
		}
		
		_, err := r.db.coll(model.CollTasks).UpdateOne(ctx,
			bson.M{"task_id": taskID},
			bson.M{
				"$set": set,
				"$push": push,
			},
		)
		return err
	}
	
	_, err := r.db.coll(model.CollTasks).UpdateOne(ctx,
		bson.M{"task_id": taskID},
		bson.M{"$set": set},
	)
	return err
}

// GetTimedOutTasks returns tasks that are running but exceeded their timeout.
func (r *TaskRepo) GetTimedOutTasks(ctx context.Context) ([]*model.Task, error) {
	now := nowUTC()
	
	// Find running tasks with timeout set and started_at + timeout_seconds < now
	cursor, err := r.db.coll(model.CollTasks).Find(ctx, bson.M{
		"status": model.TaskRunning,
		"timeout_seconds": bson.M{"$gt": 0}, // Timeout is configured
		"$expr": bson.M{
			"$lt": []interface{}{
				bson.M{"$add": []interface{}{"$started_at", bson.M{"$multiply": []interface{}{"$timeout_seconds", 1000}}}},
				now,
			},
		},
	})
	if err != nil {
		return nil, err
	}
	
	var tasks []*model.Task
	return tasks, cursor.All(ctx, &tasks)
}

// UpdateTaskStartedAt sets the started_at timestamp for a task.
func (r *TaskRepo) UpdateTaskStartedAt(ctx context.Context, taskID string) error {
	_, err := r.db.coll(model.CollTasks).UpdateOne(ctx,
		bson.M{"task_id": taskID},
		bson.M{"$set": bson.M{"started_at": nowUTC(), "updated_at": nowUTC()}},
	)
	return err
}

// DeleteByTaskID removes a task by its string UUID.
func (r *TaskRepo) DeleteByTaskID(ctx context.Context, taskID string) error {
	_, err := r.db.coll(model.CollTasks).DeleteOne(ctx, bson.M{"task_id": taskID})
	return err
}

// List returns paginated tasks with optional status filter.
func (r *TaskRepo) List(ctx context.Context, status model.TaskStatus, page, pageSize int64) ([]*model.Task, int64, error) {
	query := bson.M{}
	if status != "" {
		query["status"] = status
	}
	coll := r.db.coll(model.CollTasks)
	total, _ := coll.CountDocuments(ctx, query)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	cursor, err := coll.Find(ctx, query,
		options.Find().
			SetSort(bson.D{{Key: "created_at", Value: -1}}).
			SetSkip((page-1)*pageSize).
			SetLimit(pageSize),
	)
	if err != nil {
		return nil, 0, err
	}
	var tasks []*model.Task
	return tasks, total, cursor.All(ctx, &tasks)
}

// --------------------------------------------------------------------------
// AuditLogRepo
// --------------------------------------------------------------------------

// AuditLogRepo handles audit_logs collection.
type AuditLogRepo struct{ db *DB }

func NewAuditLogRepo(db *DB) *AuditLogRepo { return &AuditLogRepo{db: db} }

// Create inserts an audit log entry.
func (r *AuditLogRepo) Create(ctx context.Context, log *model.AuditLog) error {
	log.CreatedAt = time.Now().UTC()
	_, err := r.db.coll(model.CollAuditLogs).InsertOne(ctx, log)
	return err
}

// List returns paginated audit logs with optional user filter.
func (r *AuditLogRepo) List(ctx context.Context, userID string, page, pageSize int64) ([]*model.AuditLog, int64, error) {
	query := bson.M{}
	if userID != "" {
		if oid, err := bson.ObjectIDFromHex(userID); err == nil {
			query["user_id"] = oid
		}
	}
	coll := r.db.coll(model.CollAuditLogs)
	total, _ := coll.CountDocuments(ctx, query)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	cursor, err := coll.Find(ctx, query,
		options.Find().
			SetSort(bson.D{{Key: "created_at", Value: -1}}).
			SetSkip((page-1)*pageSize).
			SetLimit(pageSize),
	)
	if err != nil {
		return nil, 0, err
	}
	var logs []*model.AuditLog
	return logs, total, cursor.All(ctx, &logs)
}

// --------------------------------------------------------------------------
// TaskScheduleRepo
// --------------------------------------------------------------------------

// TaskScheduleRepo handles task_schedules collection.
type TaskScheduleRepo struct{ db *DB }

func NewTaskScheduleRepo(db *DB) *TaskScheduleRepo { return &TaskScheduleRepo{db: db} }

// GetByType returns the task schedule for a given type.
func (r *TaskScheduleRepo) GetByType(ctx context.Context, taskType model.TaskType) (*model.TaskSchedule, error) {
	var schedule model.TaskSchedule
	err := r.db.coll(model.CollTaskSchedules).FindOne(ctx, bson.M{"type": taskType}).Decode(&schedule)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	return &schedule, err
}

// Upsert creates or updates a task schedule.
func (r *TaskScheduleRepo) Upsert(ctx context.Context, s *model.TaskSchedule) error {
	s.UpdatedAt = nowUTC()
	filter := bson.M{"type": s.Type}
	var existing model.TaskSchedule
	err := r.db.coll(model.CollTaskSchedules).FindOne(ctx, filter).Decode(&existing)
	if errors.Is(err, mongo.ErrNoDocuments) {
		s.CreatedAt = nowUTC()
		_, err = r.db.coll(model.CollTaskSchedules).InsertOne(ctx, s)
		return err
	}
	if err != nil {
		return err
	}
	s.ID = existing.ID
	s.CreatedAt = existing.CreatedAt
	_, err = r.db.coll(model.CollTaskSchedules).UpdateOne(ctx, filter, bson.M{"$set": s})
	return err
}

// List returns all task schedules.
func (r *TaskScheduleRepo) List(ctx context.Context) ([]*model.TaskSchedule, error) {
	cursor, err := r.db.coll(model.CollTaskSchedules).Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	var schedules []*model.TaskSchedule
	return schedules, cursor.All(ctx, &schedules)
}
