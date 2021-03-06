package actions

import (
	"github.com/sensu/sensu-go/backend/authorization"
	"github.com/sensu/sensu-go/backend/store"
	"github.com/sensu/sensu-go/types"
	"golang.org/x/net/context"
)

// silencedUpdateFields whitelists fields allowed to be updated for Silences
var silencedUpdateFields = []string{
	"Expire",
	"ExpireOnResolve",
	"Reason",
	"Begin",
}

// SilencedController exposes actions in which a viewer can perform.
type SilencedController struct {
	Store  store.SilencedStore
	Policy authorization.SilencedPolicy
}

// NewSilencedController returns new SilencedController
func NewSilencedController(store store.SilencedStore) SilencedController {
	return SilencedController{
		Store:  store,
		Policy: authorization.Silenced,
	}
}

// Query returns resources available to the viewer.
func (a SilencedController) Query(ctx context.Context, params QueryParams) ([]*types.Silenced, error) {
	var results []*types.Silenced
	var serr error
	if sub := params["subscription"]; sub != "" {
		results, serr = a.Store.GetSilencedEntriesBySubscription(ctx, sub)
	} else if check := params["check"]; check != "" {
		results, serr = a.Store.GetSilencedEntriesByCheckName(ctx, check)
	} else {
		results, serr = a.Store.GetSilencedEntries(ctx)
	}
	if serr != nil {
		return nil, NewError(InternalErr, serr)
	}

	// Filter out those resources the viewer does not have access to view.
	abilities := a.Policy.WithContext(ctx)
	for i := 0; i < len(results); i++ {
		if !abilities.CanRead(results[i]) {
			results = append(results[:i], results[i+1:]...)
			i--
		}
	}

	return results, nil
}

// Find returns resource associated with given parameters if available to the
// viewer.
func (a SilencedController) Find(ctx context.Context, id string) (*types.Silenced, error) {
	// Fetch from store
	result, err := a.findSilencedEntry(ctx, id)
	if err != nil {
		return nil, err
	}

	// Verify user has permission to view
	abilities := a.Policy.WithContext(ctx)
	if result != nil && abilities.CanRead(result) {
		return result, nil
	}

	return nil, NewErrorf(NotFound)
}

// Create instatiates, validates and persists new resource if viewer has access.
func (a SilencedController) Create(ctx context.Context, newSilence types.Silenced) error {
	// Adjust context
	ctx = addOrgEnvToContext(ctx, &newSilence)
	abilities := a.Policy.WithContext(ctx)

	// Populate newSilence.ID with the subscription and checkName. Substitute a
	// splat if one of the values does not exist. If both values are empty, the
	// validator will return an error when attempting to update it in the store.
	if newSilence.Subscription != "" && newSilence.Check != "" {
		newSilence.ID = newSilence.Subscription + ":" + newSilence.Check
	} else if newSilence.Check == "" && newSilence.Subscription != "" {
		newSilence.ID = newSilence.Subscription + ":" + "*"
	} else if newSilence.Subscription == "" && newSilence.Check != "" {
		newSilence.ID = "*" + ":" + newSilence.Check
	}

	// Retrieve the subject of the JWT, which represents the logged on user, in
	// order to set it as the creator of the silenced entry
	if actor, ok := ctx.Value(types.AuthorizationActorKey).(authorization.Actor); ok {
		newSilence.Creator = actor.Name
	}

	// Validate
	if err := newSilence.Validate(); err != nil {
		return NewError(InvalidArgument, err)
	}

	// Check for existing
	if e, serr := a.Store.GetSilencedEntryByID(ctx, newSilence.ID); serr != nil {
		return NewError(InternalErr, serr)
	} else if e != nil {
		return NewErrorf(AlreadyExistsErr)
	}

	// Verify viewer can make change
	if yes := abilities.CanCreate(&newSilence); !yes {
		return NewErrorf(PermissionDenied)
	}

	// Persist
	if err := a.Store.UpdateSilencedEntry(ctx, &newSilence); err != nil {
		return NewError(InternalErr, err)
	}

	return nil
}

// Update validates and persists changes to a resource if viewer has access.
func (a SilencedController) Update(ctx context.Context, given types.Silenced) error {
	// Adjust context
	ctx = addOrgEnvToContext(ctx, &given)
	abilities := a.Policy.WithContext(ctx)

	// Find existing silenced
	// Fetch from store
	silence, err := a.findSilencedEntry(ctx, given.ID)
	if err != nil {
		return err
	}

	// Verify viewer can make change
	if yes := abilities.CanUpdate(silence); !yes {
		return NewErrorf(PermissionDenied)
	}

	// Copy
	copyFields(silence, &given, silencedUpdateFields...)

	// Validate
	if err := silence.Validate(); err != nil {
		return NewError(InvalidArgument, err)
	}

	// Persist Changes
	if serr := a.Store.UpdateSilencedEntry(ctx, silence); serr != nil {
		return NewError(InternalErr, serr)
	}

	return nil
}

// Destroy removes a resource if viewer has access.
func (a SilencedController) Destroy(ctx context.Context, params QueryParams) error {
	abilities := a.Policy.WithContext(ctx)

	// Verify user has permission
	if yes := abilities.CanDelete(); !yes {
		return NewErrorf(PermissionDenied)
	}

	// Check for ID first
	id := params["id"]

	if id == "" {
		// Form ID from subscription and check
		var err error
		sub := params["subscription"]
		check := params["check"]

		id, err = types.SilencedID(sub, check)
		if err != nil {
			return NewError(InvalidArgument, err)
		}
	}

	if err := a.Store.DeleteSilencedEntryByID(ctx, id); err != nil {
		return NewError(InternalErr, err)
	}

	return nil
}

func (a SilencedController) findSilencedEntry(ctx context.Context, id string) (*types.Silenced, error) {
	result, serr := a.Store.GetSilencedEntryByID(ctx, id)
	if serr != nil {
		return nil, NewError(InternalErr, serr)
	}
	if result != nil {
		return result, nil
	}

	return nil, NewErrorf(NotFound)
}
