package apiresource_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"go.inout.gg/foundations/apiresource"
)

type params struct {
	db *db

	// mutated, is used to track which products have been rendered
	renderedProductIDs []int
}

type db struct {
	factories map[int]*Factory
}

func (db *db) FactoryGetByID(_ context.Context, id int) (*Factory, error) {
	factory, ok := db.factories[id]
	if !ok {
		return nil, fmt.Errorf("factory %d not found", id)
	}

	return factory, nil
}

type Factory struct {
	Location string
	ID       int
}

type DBProduct struct {
	Name      string
	ID        int
	FactoryID int
}

type Product struct {
	FactoryLocation string `json:"factory_location"` //nolint:tagliatelle // API responses use snake_case.
	Name            string `json:"name"`
	ID              int    `json:"id"`
	FactoryID       int    `json:"factory_id"` //nolint:tagliatelle // API responses use snake_case.
}

func (*Product) Bundle(ctx context.Context, params *params, products []*DBProduct) (*ProductBundle, error) {
	bundle := &ProductBundle{factories: make(map[int]*Factory, len(products))}
	for _, product := range products {
		factory, err := params.db.FactoryGetByID(ctx, product.FactoryID)
		if err != nil {
			return nil, err
		}

		bundle.factories[product.FactoryID] = factory
	}

	return bundle, nil
}

func (*Product) Render(
	_ context.Context,
	params *params,
	bundle *ProductBundle,
	product *DBProduct,
) (*Product, error) {
	params.renderedProductIDs = append(params.renderedProductIDs, product.ID)

	factory := bundle.factories[product.FactoryID]
	if factory == nil {
		return nil, fmt.Errorf("factory %d not bundled", product.FactoryID)
	}

	return &Product{
		ID:              product.ID,
		FactoryID:       product.FactoryID,
		FactoryLocation: factory.Location,
		Name:            product.Name,
	}, nil
}

type ProductBundle struct {
	factories map[int]*Factory // factory ID -> factory
}

func Test_Render(t *testing.T) {
	t.Run("renders a resource", func(t *testing.T) {
		// Arrange.
		params := &params{
			db: &db{
				factories: map[int]*Factory{
					10: {ID: 10, Location: "Detroit"},
				},
			},
		}
		product := &DBProduct{ID: 1, FactoryID: 10, Name: "Pressure Gauge"}

		// Act.
		resource, err := apiresource.Render[*Product](t.Context(), params, product)

		// Assert.
		require.NoError(t, err)

		want := &Product{
			ID:              1,
			FactoryID:       10,
			FactoryLocation: "Detroit",
			Name:            "Pressure Gauge",
		}
		require.Equal(t, want, resource)
	})

	t.Run("wraps bundle errors", func(t *testing.T) {
		// Arrange.
		params := &params{db: &db{factories: map[int]*Factory{}}}
		product := &DBProduct{ID: 1, FactoryID: 10, Name: "Pressure Gauge"}

		// Act.
		resource, err := apiresource.Render[*Product](t.Context(), params, product)

		// Assert.
		require.Nil(t, resource)
		require.ErrorContains(t, err, "apiresource: error loading bundle")
		require.ErrorContains(t, err, "factory 10 not found")
	})

	t.Run("wraps render errors", func(t *testing.T) {
		// Arrange.
		params := &params{
			db: &db{factories: map[int]*Factory{10: nil}},
		}
		product := &DBProduct{ID: 1, FactoryID: 10, Name: "Pressure Gauge"}

		// Act.
		resource, err := apiresource.Render[*Product](t.Context(), params, product)

		// Assert.
		require.Nil(t, resource)
		require.ErrorContains(t, err, "apiresource: error rendering resource")
		require.ErrorContains(t, err, "factory 10 not bundled")
	})
}

func Test_RenderMany(t *testing.T) {
	t.Run("preserves input order", func(t *testing.T) {
		// Arrange.
		params := &params{
			db: &db{
				factories: map[int]*Factory{
					10: {ID: 10, Location: "Detroit"},
					20: {ID: 20, Location: "Osaka"},
				},
			},
		}
		products := []*DBProduct{
			{ID: 1, FactoryID: 10, Name: "Pressure Gauge"},
			{ID: 2, FactoryID: 20, Name: "Conveyor Motor"},
		}

		// Act.
		resources, err := apiresource.RenderMany[*Product](t.Context(), params, products)

		// Assert.
		require.NoError(t, err)

		want := []*Product{
			{ID: 1, FactoryID: 10, FactoryLocation: "Detroit", Name: "Pressure Gauge"},
			{ID: 2, FactoryID: 20, FactoryLocation: "Osaka", Name: "Conveyor Motor"},
		}
		require.Equal(t, want, resources)
	})

	t.Run("returns an empty slice", func(t *testing.T) {
		// Arrange.
		params := &params{db: &db{factories: map[int]*Factory{}}}
		products := []*DBProduct{}

		// Act.
		resources, err := apiresource.RenderMany[*Product](t.Context(), params, products)

		// Assert.
		require.NoError(t, err)
		require.NotNil(t, resources)
		require.Empty(t, resources)
	})

	t.Run("wraps bundle errors", func(t *testing.T) {
		// Arrange.
		params := &params{db: &db{factories: map[int]*Factory{}}}
		products := []*DBProduct{{ID: 1, FactoryID: 10, Name: "Pressure Gauge"}}

		// Act.
		resources, err := apiresource.RenderMany[*Product](t.Context(), params, products)

		// Assert.
		require.Nil(t, resources)
		require.ErrorContains(t, err, "apiresource: error loading bundle")
		require.ErrorContains(t, err, "factory 10 not found")
	})

	t.Run("stops on render errors", func(t *testing.T) {
		// Arrange.
		params := &params{
			db: &db{
				factories: map[int]*Factory{
					10: {ID: 10, Location: "Detroit"},
					20: nil,
					30: {ID: 30, Location: "Berlin"},
				},
			},
		}
		products := []*DBProduct{
			{ID: 1, FactoryID: 10, Name: "Pressure Gauge"},
			{ID: 2, FactoryID: 20, Name: "Conveyor Motor"},
			{ID: 3, FactoryID: 30, Name: "Hydraulic Pump"},
		}

		// Act.
		resources, err := apiresource.RenderMany[*Product](t.Context(), params, products)

		// Assert.
		require.Nil(t, resources)
		require.Equal(t, []int{1, 2}, params.renderedProductIDs)
		require.ErrorContains(t, err, "apiresource: error rendering resource")
		require.ErrorContains(t, err, "factory 20 not bundled")
	})
}

func ExampleRender() {
	params := &params{
		db: &db{
			factories: map[int]*Factory{
				10: {ID: 10, Location: "Detroit"},
			},
		},
	}
	product := &DBProduct{ID: 1, FactoryID: 10, Name: "Pressure Gauge"}

	resource, err := apiresource.Render[*Product](context.Background(), params, product)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%+v\n", resource)
	// Output: &{FactoryLocation:Detroit Name:Pressure Gauge ID:1 FactoryID:10}
}

func ExampleRenderMany() {
	params := &params{
		db: &db{
			factories: map[int]*Factory{
				10: {ID: 10, Location: "Detroit"},
				20: {ID: 20, Location: "Osaka"},
			},
		},
	}
	products := []*DBProduct{
		{ID: 1, FactoryID: 10, Name: "Pressure Gauge"},
		{ID: 2, FactoryID: 20, Name: "Conveyor Motor"},
	}

	resources, err := apiresource.RenderMany[*Product](context.Background(), params, products)
	if err != nil {
		panic(err)
	}

	for _, resource := range resources {
		fmt.Printf("%+v\n", resource)
	}

	// Output:
	// &{FactoryLocation:Detroit Name:Pressure Gauge ID:1 FactoryID:10}
	// &{FactoryLocation:Osaka Name:Conveyor Motor ID:2 FactoryID:20}
}
