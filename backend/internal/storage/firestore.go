package storage

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Firestore documents max out at ~1 MiB, so blobs are gzipped and split into
// chunk documents: storage/{name} holds the chunk count, and
// storage/{name}/chunks/{i} hold the bytes. Save runs in a transaction so a
// reader never sees a half-written snapshot.
const (
	collection = "storage"
	chunkSize  = 900_000
	opTimeout  = 30 * time.Second
)

// Firestore is a blob backend on a Firebase project's Firestore database.
type Firestore struct {
	client *firestore.Client
}

// NewFirestore connects using explicit service-account JSON (credsJSON) or,
// when empty, Application Default Credentials. projectID may be empty if the
// credentials JSON carries a project_id.
func NewFirestore(ctx context.Context, projectID string, credsJSON []byte) (*Firestore, error) {
	var opts []option.ClientOption
	if len(credsJSON) > 0 {
		opts = append(opts, option.WithCredentialsJSON(credsJSON))
		if projectID == "" {
			var creds struct {
				ProjectID string `json:"project_id"`
			}
			if err := json.Unmarshal(credsJSON, &creds); err == nil {
				projectID = creds.ProjectID
			}
		}
	}
	if projectID == "" {
		return nil, errors.New("firestore: project id missing (set FIRESTORE_PROJECT_ID)")
	}
	client, err := firestore.NewClient(ctx, projectID, opts...)
	if err != nil {
		return nil, err
	}
	return &Firestore{client: client}, nil
}

// Blob returns the named blob in this Firestore database.
func (f *Firestore) Blob(name string) Blob { return firestoreBlob{c: f.client, name: name} }

type firestoreBlob struct {
	c    *firestore.Client
	name string
}

type blobMeta struct {
	Chunks    int       `firestore:"chunks"`
	Encoding  string    `firestore:"encoding"`
	UpdatedAt time.Time `firestore:"updatedAt"`
}

func (b firestoreBlob) meta() *firestore.DocumentRef {
	return b.c.Collection(collection).Doc(b.name)
}

func (b firestoreBlob) chunk(i int) *firestore.DocumentRef {
	return b.meta().Collection("chunks").Doc(strconv.Itoa(i))
}

func (b firestoreBlob) Load() ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
	defer cancel()

	snap, err := b.meta().Get(ctx)
	if status.Code(err) == codes.NotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var meta blobMeta
	if err := snap.DataTo(&meta); err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	for i := 0; i < meta.Chunks; i++ {
		cs, err := b.chunk(i).Get(ctx)
		if err != nil {
			return nil, fmt.Errorf("chunk %d: %w", i, err)
		}
		data, err := cs.DataAt("data")
		if err != nil {
			return nil, fmt.Errorf("chunk %d: %w", i, err)
		}
		bytes, ok := data.([]byte)
		if !ok {
			return nil, fmt.Errorf("chunk %d: unexpected type %T", i, data)
		}
		buf.Write(bytes)
	}

	if meta.Encoding == "gzip" {
		zr, err := gzip.NewReader(&buf)
		if err != nil {
			return nil, err
		}
		defer zr.Close()
		return io.ReadAll(zr)
	}
	return buf.Bytes(), nil
}

func (b firestoreBlob) Save(data []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
	defer cancel()

	var zbuf bytes.Buffer
	zw := gzip.NewWriter(&zbuf)
	if _, err := zw.Write(data); err != nil {
		return err
	}
	if err := zw.Close(); err != nil {
		return err
	}
	packed := zbuf.Bytes()

	chunks := make([][]byte, 0, len(packed)/chunkSize+1)
	for len(packed) > 0 {
		n := min(len(packed), chunkSize)
		chunks = append(chunks, packed[:n])
		packed = packed[n:]
	}

	return b.c.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		// Stale chunk docs from a previously larger blob must go.
		oldChunks := 0
		if snap, err := tx.Get(b.meta()); err == nil {
			var old blobMeta
			if err := snap.DataTo(&old); err == nil {
				oldChunks = old.Chunks
			}
		} else if status.Code(err) != codes.NotFound {
			return err
		}

		if err := tx.Set(b.meta(), blobMeta{
			Chunks:    len(chunks),
			Encoding:  "gzip",
			UpdatedAt: time.Now().UTC(),
		}); err != nil {
			return err
		}
		for i, part := range chunks {
			if err := tx.Set(b.chunk(i), map[string]any{"data": part}); err != nil {
				return err
			}
		}
		for i := len(chunks); i < oldChunks; i++ {
			if err := tx.Delete(b.chunk(i)); err != nil {
				return err
			}
		}
		return nil
	})
}
