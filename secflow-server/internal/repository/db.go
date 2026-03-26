// Package repository implements MongoDB data access for all secflow models.
// Each sub-file handles one collection; this file wires the DB connection
// and shared helpers.
package repository

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/secflow/server/internal/model"
)

// DB is a thin wrapper around the mongo.Database that provides typed
// collection accessors and lifecycle management.
type DB struct {
	client *mongo.Client
	db     *mongo.Database
}

// New connects to MongoDB and returns a ready DB.
func New(ctx context.Context, uri, dbName string) (*DB, error) {
	opts := options.Client().ApplyURI(uri).
		SetConnectTimeout(10 * time.Second).
		SetServerSelectionTimeout(10 * time.Second)

	client, err := mongo.Connect(opts)
	if err != nil {
		return nil, fmt.Errorf("mongo connect: %w", err)
	}
	if err = client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("mongo ping: %w", err)
	}
	d := &DB{client: client, db: client.Database(dbName)}
	if err = d.ensureIndexes(ctx); err != nil {
		return nil, err
	}
	return d, nil
}

// Close terminates the MongoDB connection.
func (d *DB) Close(ctx context.Context) error {
	return d.client.Disconnect(ctx)
}

// coll returns a typed collection handle.
func (d *DB) coll(name string) *mongo.Collection {
	return d.db.Collection(name)
}

// Collection is the public version of coll, for use by handlers that need
// direct collection access (e.g., when no repo method exists yet).
func (d *DB) Collection(name string) *mongo.Collection {
	return d.db.Collection(name)
}

// ensureIndexes creates all required MongoDB indexes on startup.
func (d *DB) ensureIndexes(ctx context.Context) error {
	type idxDef struct {
		coll    string
		keys    bson.D
		unique  bool
		sparse  bool
	}
	defs := []idxDef{
		// users
		{model.CollUsers, bson.D{{Key: "username", Value: 1}}, true, false},
		{model.CollUsers, bson.D{{Key: "email", Value: 1}}, true, false},
		// invite_codes
		{model.CollInviteCodes, bson.D{{Key: "code", Value: 1}}, true, false},
		{model.CollInviteCodes, bson.D{{Key: "owner_id", Value: 1}}, false, false},
		// nodes
		{model.CollNodes, bson.D{{Key: "node_id", Value: 1}}, true, false},
		// tasks
		{model.CollTasks, bson.D{{Key: "task_id", Value: 1}}, true, false},
		{model.CollTasks, bson.D{{Key: "status", Value: 1}}, false, false},
		{model.CollTasks, bson.D{{Key: "assigned_to", Value: 1}}, false, true},
		// vuln_records
		{model.CollVulnRecords, bson.D{{Key: "key", Value: 1}}, true, false},
		{model.CollVulnRecords, bson.D{{Key: "cve", Value: 1}}, false, true},
		{model.CollVulnRecords, bson.D{{Key: "pushed", Value: 1}}, false, false},
		{model.CollVulnRecords, bson.D{{Key: "severity", Value: 1}}, false, false},
		// articles
		{model.CollArticles, bson.D{{Key: "url", Value: 1}}, true, false},
		{model.CollArticles, bson.D{{Key: "source", Value: 1}}, false, false},
		// audit_logs
		{model.CollAuditLogs, bson.D{{Key: "user_id", Value: 1}}, false, false},
		{model.CollAuditLogs, bson.D{{Key: "created_at", Value: -1}}, false, false},
		// reports
		{model.CollReports, bson.D{{Key: "created_by", Value: 1}}, false, false},
	}

	for _, def := range defs {
		idxModel := mongo.IndexModel{
			Keys: def.keys,
			Options: options.Index().
				SetUnique(def.unique).
				SetSparse(def.sparse),
		}
		// Drop existing index with same name but different options to avoid conflicts
		// Get the auto-generated index name and drop it first
		indexName := fmt.Sprintf("%s_%s", def.keys[0].Key, def.keys[0].Value)
		d.coll(def.coll).Indexes().DropOne(ctx, indexName)

		if _, err := d.coll(def.coll).Indexes().CreateOne(ctx, idxModel); err != nil {
			return fmt.Errorf("create index on %s: %w", def.coll, err)
		}
	}
	return nil
}

// nowUTC is a convenience helper.
func nowUTC() time.Time { return time.Now().UTC() }
