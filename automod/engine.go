package automod

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	appbsky "github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/bluesky-social/indigo/xrpc"
)

// runtime for executing rules, managing state, and recording moderation actions.
//
// TODO: careful when initializing: several fields should not be null or zero, even though they are pointer type.
type Engine struct {
	Logger  *slog.Logger
	Directory identity.Directory
	// current rule sets. will eventually be possible to swap these out at runtime
	RulesMap  sync.Map
	// used to persist moderation actions in mod service (optional)
	AdminClient *xrpc.Client
	CountStore  CountStore
}

func (e *Engine) ProcessIdentityEvent(ctx context.Context, t string, did syntax.DID) error {
	// similar to an HTTP server, we want to recover any panics from rule execution
	defer func() {
		if r := recover(); r != nil {
			e.Logger.Error("automod event execution exception", "err", r)
		}
	}()

	ident, err := e.Directory.LookupDID(ctx, did)
	if err != nil {
		return fmt.Errorf("resolving identity: %w", err)
	}
	if ident == nil {
		return fmt.Errorf("identity not found for did: %s", did.String())
	}

	evt := IdentityEvent{
		Event{
			Engine:  e,
			Account: AccountMeta{Identity: ident},
		},
	}
	// TODO: call rules
	_ = evt
	return nil
}

func (e *Engine) ProcessRecord(ctx context.Context, did syntax.DID, path string, rec any) error {
	// similar to an HTTP server, we want to recover any panics from rule execution
	defer func() {
		if r := recover(); r != nil {
			e.Logger.Error("automod event execution exception", "err", r)
		}
	}()

	ident, err := e.Directory.LookupDID(ctx, did)
	if err != nil {
		return fmt.Errorf("resolving identity: %w", err)
	}
	if ident == nil {
		return fmt.Errorf("identity not found for did: %s", did.String())
	}
	collection := strings.SplitN(path, "/", 2)[0]

	switch collection {
	case "app.bsky.feed.post":
		post, ok := rec.(*appbsky.FeedPost)
		if !ok {
			return fmt.Errorf("mismatch between collection (%s) and type", collection)
		}
		evt := e.NewPostEvent(ident, path, post)
		e.Logger.Info("processing post", "did", ident.DID, "path", path)
		_ = evt
		// TODO: call rules
	default:
		evt := e.NewRecordEvent(ident, path, rec)
		e.Logger.Info("processing record", "did", ident.DID, "path", path)
		_ = evt
		// TODO: call rules
	}

	return nil
}

func (e *Engine) NewPostEvent(ident *identity.Identity, path string, post *appbsky.FeedPost) PostEvent {
	return PostEvent{
		RecordEvent {
			Event{
				Engine:  e,
				Account: AccountMeta{Identity: ident},
			},
			[]string{},
			false,
			[]ModReport{},
			[]string{},
		},
	}
}

func (e *Engine) NewRecordEvent(ident *identity.Identity, path string, rec any) RecordEvent {
	return RecordEvent{
		Event{
			Engine:  e,
			Account: AccountMeta{Identity: ident},
		},
		[]string{},
		false,
		[]ModReport{},
		[]string{},
	}
}

func (e *Engine) GetCount(key, period string) (int, error) {
	return e.CountStore.GetCount(context.TODO(), key, period)
}

func (e *Engine) InSet(setName, val string) (bool, error) {
	// XXX: implement
	return false, nil
}
