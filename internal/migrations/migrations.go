package migrations

import (
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// Register sets up database migrations for the stats tracking schema
func Register(app *pocketbase.PocketBase) {
	// Create collections on app bootstrap
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		// Create sites collection (registered domains)
		if err := createSitesCollection(app); err != nil {
			return err
		}

		// Migrate existing sites collection to add new fields
		if err := migrateSitesCollection(app); err != nil {
			return err
		}

		// Create pageviews collection (analytics data)
		if err := createPageviewsCollection(app); err != nil {
			return err
		}

		// Create denied_pageviews collection (tracking denied requests)
		if err := createDeniedPageviewsCollection(app); err != nil {
			return err
		}

		return e.Next()
	})
}

// createSitesCollection creates the sites collection for registered domains
func createSitesCollection(app *pocketbase.PocketBase) error {
	// Check if collection already exists
	existing, _ := app.FindCollectionByNameOrId("sites")
	if existing != nil {
		return nil
	}

	collection := core.NewBaseCollection("sites")

	// Set rules - empty string means public access, nil means admin only
	collection.ListRule = nil
	collection.ViewRule = nil
	collection.CreateRule = nil
	collection.UpdateRule = nil
	collection.DeleteRule = nil

	// Add fields
	collection.Fields.Add(&core.TextField{
		Name:     "domain",
		Required: true,
		Max:      255,
	})

	collection.Fields.Add(&core.TextField{
		Name:     "name",
		Required: true,
		Max:      255,
	})

	collection.Fields.Add(&core.BoolField{
		Name: "active",
	})

	collection.Fields.Add(&core.TextField{
		Name: "additional_domains",
		Max:  1024,
	})

	// Add index
	collection.AddIndex("idx_sites_domain", false, "domain", "")

	return app.Save(collection)
}

// migrateSitesCollection adds new fields to existing sites collection
func migrateSitesCollection(app *pocketbase.PocketBase) error {
	collection, err := app.FindCollectionByNameOrId("sites")
	if err != nil {
		return nil // Collection doesn't exist, nothing to migrate
	}

	// Check if additional_domains field already exists
	if collection.Fields.GetByName("additional_domains") != nil {
		return nil // Already migrated
	}

	// Add the additional_domains field
	collection.Fields.Add(&core.TextField{
		Name: "additional_domains",
		Max:  1024,
	})

	return app.Save(collection)
}

// createPageviewsCollection creates the pageviews collection for analytics
func createPageviewsCollection(app *pocketbase.PocketBase) error {
	// Check if collection already exists
	existing, _ := app.FindCollectionByNameOrId("pageviews")
	if existing != nil {
		return nil
	}

	// Look up the sites collection to get its ID
	sitesCollection, err := app.FindCollectionByNameOrId("sites")
	if err != nil {
		return err
	}

	collection := core.NewBaseCollection("pageviews")

	// Set rules
	collection.ListRule = nil
	collection.ViewRule = nil
	collection.CreateRule = nil
	collection.UpdateRule = nil
	collection.DeleteRule = nil

	// Add fields
	collection.Fields.Add(&core.RelationField{
		Name:          "site",
		Required:      true,
		CollectionId:  sitesCollection.Id,
		MaxSelect:     1,
		CascadeDelete: true,
	})

	collection.Fields.Add(&core.TextField{
		Name:     "path",
		Required: true,
		Max:      2048,
	})

	collection.Fields.Add(&core.TextField{
		Name: "referrer",
		Max:  2048,
	})

	collection.Fields.Add(&core.TextField{
		Name: "user_agent",
		Max:  1024,
	})

	collection.Fields.Add(&core.TextField{
		Name: "ip_hash",
		Max:  64,
	})

	collection.Fields.Add(&core.TextField{
		Name: "country",
		Max:  64,
	})

	collection.Fields.Add(&core.NumberField{
		Name: "screen_width",
	})

	collection.Fields.Add(&core.NumberField{
		Name: "screen_height",
	})

	// Add indexes
	collection.AddIndex("idx_pageviews_site", false, "site", "")
	collection.AddIndex("idx_pageviews_created", false, "created", "")
	collection.AddIndex("idx_pageviews_path", false, "path", "")

	return app.Save(collection)
}

// createDeniedPageviewsCollection creates a collection to track denied/unauthorized requests
func createDeniedPageviewsCollection(app *pocketbase.PocketBase) error {
	// Check if collection already exists
	existing, _ := app.FindCollectionByNameOrId("denied_pageviews")
	if existing != nil {
		return nil
	}

	collection := core.NewBaseCollection("denied_pageviews")

	// Admin only access
	collection.ListRule = nil
	collection.ViewRule = nil
	collection.CreateRule = nil
	collection.UpdateRule = nil
	collection.DeleteRule = nil

	// Add fields
	collection.Fields.Add(&core.TextField{
		Name:     "domain",
		Required: true,
		Max:      255,
	})

	collection.Fields.Add(&core.TextField{
		Name:     "origin",
		Required: true,
		Max:      2048,
	})

	collection.Fields.Add(&core.TextField{
		Name:     "reason",
		Required: true,
		Max:      255,
	})

	collection.Fields.Add(&core.TextField{
		Name: "path",
		Max:  2048,
	})

	collection.Fields.Add(&core.TextField{
		Name: "referrer",
		Max:  2048,
	})

	collection.Fields.Add(&core.TextField{
		Name: "user_agent",
		Max:  1024,
	})

	collection.Fields.Add(&core.TextField{
		Name: "ip_hash",
		Max:  64,
	})

	collection.Fields.Add(&core.NumberField{
		Name: "screen_width",
	})

	collection.Fields.Add(&core.NumberField{
		Name: "screen_height",
	})

	// Add indexes (note: 'created' is a system field, can't index before save)
	collection.AddIndex("idx_denied_domain", false, "domain", "")
	collection.AddIndex("idx_denied_reason", false, "reason", "")

	return app.Save(collection)
}
