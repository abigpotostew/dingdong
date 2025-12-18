package migrations

import (
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/models/schema"
	"github.com/pocketbase/pocketbase/tools/types"
)

// Register sets up database migrations for the stats tracking schema
func Register(app *pocketbase.PocketBase) {
	// Create collections on app bootstrap
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		// Create sites collection (registered domains)
		if err := createSitesCollection(app); err != nil {
			return err
		}

		// Create pageviews collection (analytics data)
		if err := createPageviewsCollection(app); err != nil {
			return err
		}

		return nil
	})
}

// createSitesCollection creates the sites collection for registered domains
func createSitesCollection(app *pocketbase.PocketBase) error {
	// Check if collection already exists
	existing, _ := app.Dao().FindCollectionByNameOrId("sites")
	if existing != nil {
		return nil
	}

	collection := &models.Collection{
		Name:       "sites",
		Type:       models.CollectionTypeBase,
		ListRule:   types.Pointer(""),
		ViewRule:   types.Pointer(""),
		CreateRule: nil, // Only admin can create
		UpdateRule: nil, // Only admin can update
		DeleteRule: nil, // Only admin can delete
		Schema: schema.NewSchema(
			&schema.SchemaField{
				Name:     "domain",
				Type:     schema.FieldTypeText,
				Required: true,
				Options: &schema.TextOptions{
					Min: types.Pointer(1),
					Max: types.Pointer(255),
				},
			},
			&schema.SchemaField{
				Name:     "name",
				Type:     schema.FieldTypeText,
				Required: true,
				Options: &schema.TextOptions{
					Min: types.Pointer(1),
					Max: types.Pointer(255),
				},
			},
			&schema.SchemaField{
				Name:     "active",
				Type:     schema.FieldTypeBool,
				Required: false,
			},
		),
		Indexes: types.JsonArray[string]{
			"CREATE INDEX idx_sites_domain ON sites (domain)",
		},
	}

	return app.Dao().SaveCollection(collection)
}

// createPageviewsCollection creates the pageviews collection for analytics
func createPageviewsCollection(app *pocketbase.PocketBase) error {
	// Check if collection already exists
	existing, _ := app.Dao().FindCollectionByNameOrId("pageviews")
	if existing != nil {
		return nil
	}

	// Look up the sites collection to get its ID
	sitesCollection, err := app.Dao().FindCollectionByNameOrId("sites")
	if err != nil {
		return err
	}

	collection := &models.Collection{
		Name:       "pageviews",
		Type:       models.CollectionTypeBase,
		ListRule:   types.Pointer(""),
		ViewRule:   types.Pointer(""),
		CreateRule: types.Pointer(""), // Public can create (via API)
		UpdateRule: nil,               // No updates
		DeleteRule: nil,               // Only admin can delete
		Schema: schema.NewSchema(
			&schema.SchemaField{
				Name:     "site",
				Type:     schema.FieldTypeRelation,
				Required: true,
				Options: &schema.RelationOptions{
					CollectionId:  sitesCollection.Id,
					MaxSelect:     types.Pointer(1),
					CascadeDelete: true,
				},
			},
			&schema.SchemaField{
				Name:     "path",
				Type:     schema.FieldTypeText,
				Required: true,
				Options: &schema.TextOptions{
					Max: types.Pointer(2048),
				},
			},
			&schema.SchemaField{
				Name:     "referrer",
				Type:     schema.FieldTypeText,
				Required: false,
				Options: &schema.TextOptions{
					Max: types.Pointer(2048),
				},
			},
			&schema.SchemaField{
				Name:     "user_agent",
				Type:     schema.FieldTypeText,
				Required: false,
				Options: &schema.TextOptions{
					Max: types.Pointer(1024),
				},
			},
			&schema.SchemaField{
				Name:     "ip_hash",
				Type:     schema.FieldTypeText,
				Required: false,
				Options: &schema.TextOptions{
					Max: types.Pointer(64),
				},
			},
			&schema.SchemaField{
				Name:     "country",
				Type:     schema.FieldTypeText,
				Required: false,
				Options: &schema.TextOptions{
					Max: types.Pointer(64),
				},
			},
			&schema.SchemaField{
				Name:     "screen_width",
				Type:     schema.FieldTypeNumber,
				Required: false,
			},
			&schema.SchemaField{
				Name:     "screen_height",
				Type:     schema.FieldTypeNumber,
				Required: false,
			},
		),
		Indexes: types.JsonArray[string]{
			"CREATE INDEX idx_pageviews_site ON pageviews (site)",
			"CREATE INDEX idx_pageviews_created ON pageviews (created)",
			"CREATE INDEX idx_pageviews_path ON pageviews (path)",
		},
	}

	return app.Dao().SaveCollection(collection)
}
