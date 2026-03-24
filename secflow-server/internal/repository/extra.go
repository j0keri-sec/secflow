package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/secflow/server/internal/model"
)

// ── Article ───────────────────────────────────────────────────────────────

// ArticleRepository handles all articles collection operations.
type ArticleRepository struct{ db *DB }

// NewArticleRepository creates a new ArticleRepository.
func NewArticleRepository(db *DB) *ArticleRepository { return &ArticleRepository{db: db} }

// List returns a paginated, filtered list of articles.
func (r *ArticleRepository) List(ctx context.Context, filter bson.D, page, pageSize int) ([]*model.Article, int64, error) {
	coll := r.db.coll(model.CollArticles)
	opts := options.Find().
		SetSort(bson.D{{Key: "published_at", Value: -1}}).
		SetSkip(int64((page - 1) * pageSize)).
		SetLimit(int64(pageSize))

	total, err := coll.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	cursor, err := coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var items []*model.Article
	if err := cursor.All(ctx, &items); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetByID returns a single article by ObjectID.
func (r *ArticleRepository) GetByID(ctx context.Context, id bson.ObjectID) (*model.Article, error) {
	coll := r.db.coll(model.CollArticles)
	var a model.Article
	err := coll.FindOne(ctx, bson.M{"_id": id}).Decode(&a)
	if err == mongo.ErrNoDocuments {
		return nil, ErrNotFound
	}
	return &a, err
}

// Delete removes a single article by ObjectID.
func (r *ArticleRepository) Delete(ctx context.Context, id bson.ObjectID) error {
	_, err := r.db.coll(model.CollArticles).DeleteOne(ctx, bson.M{"_id": id})
	return err
}

// BulkUpsert inserts or updates articles by URL (unique key).
func (r *ArticleRepository) BulkUpsert(ctx context.Context, items []*model.Article) (int, error) {
	coll := r.db.coll(model.CollArticles)
	upserted := 0
	for _, a := range items {
		if a.CreatedAt.IsZero() {
			a.CreatedAt = time.Now()
		}
		res, err := coll.UpdateOne(ctx,
			bson.M{"url": a.URL},
			bson.M{"$setOnInsert": a},
			options.UpdateOne().SetUpsert(true),
		)
		if err != nil {
			return upserted, err
		}
		if res.UpsertedCount > 0 {
			upserted++
		}
	}
	return upserted, nil
}

// ── PushChannel ───────────────────────────────────────────────────────────

// PushChannelRepository manages push_channels collection.
type PushChannelRepository struct{ db *DB }

// NewPushChannelRepository creates a new PushChannelRepository.
func NewPushChannelRepository(db *DB) *PushChannelRepository {
	return &PushChannelRepository{db: db}
}

// List returns all push channels.
func (r *PushChannelRepository) List(ctx context.Context) ([]*model.PushChannel, error) {
	cursor, err := r.db.coll(model.CollPushChannels).Find(ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var items []*model.PushChannel
	return items, cursor.All(ctx, &items)
}

// Create inserts a new push channel.
func (r *PushChannelRepository) Create(ctx context.Context, ch *model.PushChannel) error {
	res, err := r.db.coll(model.CollPushChannels).InsertOne(ctx, ch)
	if err != nil {
		return err
	}
	ch.ID = res.InsertedID.(bson.ObjectID)
	return nil
}

// Update applies a partial update to a push channel.
func (r *PushChannelRepository) Update(ctx context.Context, id bson.ObjectID, patch map[string]any) error {
	_, err := r.db.coll(model.CollPushChannels).UpdateOne(ctx,
		bson.M{"_id": id},
		bson.M{"$set": patch},
	)
	return err
}

// Delete removes a push channel.
func (r *PushChannelRepository) Delete(ctx context.Context, id bson.ObjectID) error {
	_, err := r.db.coll(model.CollPushChannels).DeleteOne(ctx, bson.M{"_id": id})
	return err
}

// ── AuditLog ──────────────────────────────────────────────────────────────

// AuditLogRepository manages audit_logs collection.
type AuditLogRepository struct{ db *DB }

// NewAuditLogRepository creates a new AuditLogRepository.
func NewAuditLogRepository(db *DB) *AuditLogRepository { return &AuditLogRepository{db: db} }

// Insert adds a new audit log entry.
func (r *AuditLogRepository) Insert(ctx context.Context, log *model.AuditLog) error {
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now()
	}
	_, err := r.db.coll(model.CollAuditLogs).InsertOne(ctx, log)
	return err
}

// List returns paginated audit log entries.
func (r *AuditLogRepository) List(ctx context.Context, filter bson.D, page, pageSize int) ([]*model.AuditLog, int64, error) {
	coll := r.db.coll(model.CollAuditLogs)
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetSkip(int64((page - 1) * pageSize)).
		SetLimit(int64(pageSize))

	total, err := coll.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	cursor, err := coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var items []*model.AuditLog
	return items, total, cursor.All(ctx, &items)
}

// ── Report ────────────────────────────────────────────────────────────────

// ReportRepository manages reports collection.
type ReportRepository struct{ db *DB }

// NewReportRepository creates a new ReportRepository.
func NewReportRepository(db *DB) *ReportRepository { return &ReportRepository{db: db} }

// Create inserts a new report.
func (r *ReportRepository) Create(ctx context.Context, report *model.Report) error {
	res, err := r.db.coll(model.CollReports).InsertOne(ctx, report)
	if err != nil {
		return err
	}
	report.ID = res.InsertedID.(bson.ObjectID)
	return nil
}

// List returns paginated reports.
func (r *ReportRepository) List(ctx context.Context, filter bson.D, page, pageSize int) ([]*model.Report, int64, error) {
	coll := r.db.coll(model.CollReports)
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetSkip(int64((page - 1) * pageSize)).
		SetLimit(int64(pageSize))

	total, err := coll.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	cursor, err := coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var items []*model.Report
	return items, total, cursor.All(ctx, &items)
}

// Update applies a partial update to a report.
func (r *ReportRepository) Update(ctx context.Context, id bson.ObjectID, patch map[string]any) error {
	_, err := r.db.coll(model.CollReports).UpdateOne(ctx,
		bson.M{"_id": id},
		bson.M{"$set": patch},
	)
	return err
}

// Delete removes a report.
func (r *ReportRepository) Delete(ctx context.Context, id bson.ObjectID) error {
	_, err := r.db.coll(model.CollReports).DeleteOne(ctx, bson.M{"_id": id})
	return err
}
