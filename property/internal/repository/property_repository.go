package repository

import (
	"context"
	"database/sql"

	db "github.com/demola234/property/db/sqlc"
	"github.com/demola234/property/internal/domain/entity"
	shopspring "github.com/shopspring/decimal"

	"github.com/google/uuid"
)

type PropertyRepository struct {
	store db.Store
}

func NewPropertyRepository(store db.Store) *PropertyRepository {
	return &PropertyRepository{
		store: store,
	}
}

func (r *PropertyRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.Property, error) {
	property, err := r.store.GetPropertyByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return r.mapSQLCPropertyToEntity(property), nil
}

func (r *PropertyRepository) GetWithAllDetails(ctx context.Context, id uuid.UUID) (*entity.PropertyWithDetails, error) {
	result, err := r.store.GetPropertyWithAllDetails(ctx, id)
	if err != nil {
		return nil, err
	}
	return r.mapSQLCPropertyWithDetailsToEntity(result), nil
}

func (r *PropertyRepository) Create(ctx context.Context, property *entity.Property) error {
	params := db.CreatePropertyParams{
		Title:       property.Title,
		Description: sql.NullString{String: *property.Description, Valid: property.Description != nil},
		Price:       shopspring.RequireFromString(property.Price.Value),
		Category:    db.PropertyCategory(property.Category),
		Type:        db.PropertyType(property.Type),
		Address:     property.Address,
		City:        property.City,
		State:       property.State,
		Country:     property.Country,
		ZipCode:     sql.NullString{String: *property.ZipCode, Valid: property.ZipCode != nil},
		OwnerID:     toNullUUID(property.OwnerID),
		Status:      db.PropertyStatus(property.Status),
	}

	created, err := r.store.CreateProperty(ctx, params)
	if err != nil {
		return err
	}

	property.ID = created.ID
	property.CreatedAt = toTime(created.CreatedAt)
	property.UpdatedAt = toTime(created.UpdatedAt)
	return nil
}

func (r *PropertyRepository) Update(ctx context.Context, property *entity.Property) error {
	params := db.UpdatePropertyParams{
		ID:          property.ID,
		Title:       property.Title,
		Description: sql.NullString{String: *property.Description, Valid: property.Description != nil},
		Category:    db.PropertyCategory(property.Category),
		Type:        db.PropertyType(property.Type),
		Address:     property.Address,
		City:        property.City,
		State:       property.State,
		Country:     property.Country,
		ZipCode:     sql.NullString{String: *property.ZipCode, Valid: property.ZipCode != nil},
		Status:      db.PropertyStatus(property.Status),
	}

	_, err := r.store.UpdateProperty(ctx, params)
	if err != nil {
		return err
	}

	property.UpdatedAt = sql.NullTime{Time: property.UpdatedAt, Valid: true}.Time
	return nil
}

func (r *PropertyRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status entity.PropertyStatus) error {
	_, err := r.store.UpdatePropertyStatus(ctx, db.UpdatePropertyStatusParams{
		ID:     id,
		Status: db.PropertyStatus(status),
	})
	if err != nil {
		return err
	}
	return nil
}

func (r *PropertyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	err := r.store.DeleteProperty(ctx, id)
	return err
}

func (r *PropertyRepository) List(ctx context.Context, filter entity.PropertySearchFilter) ([]*entity.Property, error) {
	params := db.ListPropertiesParams{
		Category:     toNullPropertyCategory(filter.Category),
		Type:         toNullPropertyType(filter.Type),
		Status:       toNullPropertyStatus(filter.Status),
		City:         sql.NullString{String: derefString(filter.City), Valid: filter.City != nil},
		State:        sql.NullString{String: derefString(filter.State), Valid: filter.State != nil},
		Country:      sql.NullString{String: derefString(filter.Country), Valid: filter.Country != nil},
		MinPrice:     sql.NullString{String: protoDecimalValue(filter.MinPrice), Valid: filter.MinPrice != nil},
		MaxPrice:     sql.NullString{String: protoDecimalValue(filter.MaxPrice), Valid: filter.MaxPrice != nil},
		MinBedrooms:  sql.NullInt32{Int32: derefInt32(filter.MinBedrooms), Valid: filter.MinBedrooms != nil},
		MinBathrooms: sql.NullInt32{Int32: derefInt32(filter.MinBathrooms), Valid: filter.MinBathrooms != nil},
		Limit:        filter.Limit,
		Offset:       filter.Offset,
	}

	properties, err := r.store.ListProperties(ctx, params)
	if err != nil {
		return nil, err
	}

	result := make([]*entity.Property, len(properties))
	for i, p := range properties {
		result[i] = r.mapSQLCListPropertyToEntity(p)
	}

	return result, nil
}

func (r *PropertyRepository) SearchWithDetails(ctx context.Context, filter entity.PropertySearchFilter) ([]*entity.PropertyWithDetails, error) {
	params := db.SearchPropertiesWithDetailsParams{
		Column1:  db.PropertyCategory(derefPropertyCategory(filter.Category)),
		Column2:  db.PropertyType(derefPropertyType(filter.Type)),
		Column3:  db.PropertyStatus(derefPropertyStatus(filter.Status)),
		Column4:  derefString(filter.City),
		Column5:  derefString(filter.State),
		Column6:  derefString(filter.Country),
		Column7:  shopspring.RequireFromString(protoDecimalValueOrZero(filter.MinPrice)),
		Column8:  shopspring.RequireFromString(protoDecimalValueOrZero(filter.MaxPrice)),
		Column9:  derefInt32(filter.MinBedrooms),
		Column10: derefInt32(filter.MinBathrooms),
		Column11: derefInt32(filter.MinSquareFeet),
		Column12: derefInt32(filter.MinYearBuilt),
		Column13: derefInt32(filter.MinGarageCount),
		Column14: derefBool(filter.HasBasement),
		Column15: derefBool(filter.HasAttic),
		Column16: derefInt32(filter.MinWalkScore),
		Column17: derefInt32(filter.MinSchoolRating),
		Column18: nil,
		Limit:    filter.Limit,
		Offset:   filter.Offset,
	}

	rows, err := r.store.SearchPropertiesWithDetails(ctx, params)
	if err != nil {
		return nil, err
	}

	result := make([]*entity.PropertyWithDetails, len(rows))
	for i, p := range rows {
		result[i] = r.mapSQLCSearchResultToEntity(p)
	}
	return result, nil
}

func (r *PropertyRepository) Count(ctx context.Context, filter entity.PropertySearchFilter) (int64, error) {
	params := db.CountPropertiesParams{
		Column1: db.PropertyCategory(derefPropertyCategory(filter.Category)),
		Column2: db.PropertyType(derefPropertyType(filter.Type)),
		Column3: db.PropertyStatus(derefPropertyStatus(filter.Status)),
		Column4: derefString(filter.City),
		Column5: derefString(filter.State),
		Column6: derefString(filter.Country),
		Column7: shopspring.RequireFromString(protoDecimalValueOrZero(filter.MinPrice)),
		Column8: shopspring.RequireFromString(protoDecimalValueOrZero(filter.MaxPrice)),
	}
	return r.store.CountProperties(ctx, params)
}

func (r *PropertyRepository) GetByOwner(ctx context.Context, ownerID uuid.UUID, limit, offset int32) ([]*entity.Property, error) {
	properties, err := r.store.GetPropertiesByOwner(ctx, db.GetPropertiesByOwnerParams{
		OwnerID: uuid.NullUUID{UUID: ownerID, Valid: true},
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		return nil, err
	}

	result := make([]*entity.Property, len(properties))
	for i, p := range properties {
		result[i] = r.mapSQLCPropertyToEntity(p)
	}
	return result, nil
}

func (r *PropertyRepository) GetByAmenity(ctx context.Context, amenityID uuid.UUID, status *entity.PropertyStatus, limit, offset int32) ([]*entity.Property, error) {
	var dbStatus db.PropertyStatus
	if status != nil {
		dbStatus = db.PropertyStatus(*status)
	}

	properties, err := r.store.GetPropertiesByAmenity(ctx, db.GetPropertiesByAmenityParams{
		ID:      amenityID,
		Column2: dbStatus,
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		return nil, err
	}

	result := make([]*entity.Property, len(properties))
	for i, p := range properties {
		result[i] = r.mapSQLCPropertyToEntity(p)
	}
	return result, nil
}

func (r *PropertyRepository) GetByMultipleAmenities(ctx context.Context, amenityIDs []uuid.UUID, status *entity.PropertyStatus, limit, offset int32) ([]*entity.Property, error) {
	var dbStatus db.PropertyStatus
	if status != nil {
		dbStatus = db.PropertyStatus(*status)
	}

	rows, err := r.store.GetPropertiesByMultipleAmenities(ctx, db.GetPropertiesByMultipleAmenitiesParams{
		Column1: amenityIDs,
		Column2: dbStatus,
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		return nil, err
	}

	result := make([]*entity.Property, len(rows))
	for i, p := range rows {
		result[i] = &entity.Property{
			ID:        p.ID,
			Title:     p.Title,
			Category:  entity.PropertyCategory(p.Category),
			Type:      entity.PropertyType(p.Type),
			Address:   p.Address,
			City:      p.City,
			State:     p.State,
			Country:   p.Country,
			OwnerID:   toNullUUIDPtr(p.OwnerID),
			Status:    entity.PropertyStatus(p.Status),
			CreatedAt: toTime(p.CreatedAt),
			UpdatedAt: toTime(p.UpdatedAt),
		}
	}
	return result, nil
}

// --- mapping helpers ---

func (r *PropertyRepository) mapSQLCPropertyToEntity(p db.Property) *entity.Property {
	return &entity.Property{
		ID:          p.ID,
		Title:       p.Title,
		Description: toNullStringPtr(p.Description),
		Category:    entity.PropertyCategory(p.Category),
		Type:        entity.PropertyType(p.Type),
		Address:     p.Address,
		City:        p.City,
		State:       p.State,
		Country:     p.Country,
		ZipCode:     toNullStringPtr(p.ZipCode),
		OwnerID:     toNullUUIDPtr(p.OwnerID),
		Status:      entity.PropertyStatus(p.Status),
		CreatedAt:   toTime(p.CreatedAt),
		UpdatedAt:   toTime(p.UpdatedAt),
	}
}

func (r *PropertyRepository) mapSQLCListPropertyToEntity(p db.ListPropertiesRow) *entity.Property {
	return &entity.Property{
		ID:          p.ID,
		Title:       p.Title,
		Description: toNullStringPtr(p.Description),
		Category:    entity.PropertyCategory(p.Category),
		Type:        entity.PropertyType(p.Type),
		Address:     p.Address,
		City:        p.City,
		State:       p.State,
		Country:     p.Country,
		ZipCode:     toNullStringPtr(p.ZipCode),
		OwnerID:     toNullUUIDPtr(p.OwnerID),
		Status:      entity.PropertyStatus(p.Status),
		CreatedAt:   toTime(p.CreatedAt),
		UpdatedAt:   toTime(p.UpdatedAt),
	}
}

func (r *PropertyRepository) mapSQLCPropertyWithDetailsToEntity(p db.GetPropertyWithAllDetailsRow) *entity.PropertyWithDetails {
	return &entity.PropertyWithDetails{
		Property: entity.Property{
			ID:          p.ID,
			Title:       p.Title,
			Description: toNullStringPtr(p.Description),
			Category:    entity.PropertyCategory(p.Category),
			Type:        entity.PropertyType(p.Type),
			Address:     p.Address,
			City:        p.City,
			State:       p.State,
			Country:     p.Country,
			ZipCode:     toNullStringPtr(p.ZipCode),
			OwnerID:     toNullUUIDPtr(p.OwnerID),
			Status:      entity.PropertyStatus(p.Status),
			CreatedAt:   toTime(p.CreatedAt),
			UpdatedAt:   toTime(p.UpdatedAt),
		},
		PropertyDetail: &entity.PropertyDetail{
			ID:               p.ID_2.UUID,
			PropertyID:       p.PropertyID.UUID,
			Bedrooms:         toNullInt32Ptr(p.Bedrooms),
			Bathrooms:        toNullInt32Ptr(p.Bathrooms),
			Toilets:          toNullInt32Ptr(p.Toilets),
			SquareFootage:    toProtoDecimal(p.SquareFootage),
			LotSize:          toProtoDecimal(p.LotSize),
			YearBuilt:        toNullInt32Ptr(p.YearBuilt),
			Stories:          toNullInt32Ptr(p.Stories),
			GarageCount:      toNullInt32Ptr(p.GarageCount),
			HasBasement:      p.HasBasement.Bool,
			HasAttic:         p.HasAttic.Bool,
			HeatingSystem:    toNullStringPtr(p.HeatingSystem),
			CoolingSystem:    toNullStringPtr(p.CoolingSystem),
			WaterSource:      toNullStringPtr(p.WaterSource),
			SewerType:        toNullStringPtr(p.SewerType),
			RoofType:         toNullStringPtr(p.RoofType),
			ExteriorMaterial: toNullStringPtr(p.ExteriorMaterial),
			FoundationType:   toNullStringPtr(p.FoundationType),
			PoolType:         toNullStringPtr(p.PoolType),
			CreatedAt:        p.CreatedAt_2.Time,
			UpdatedAt:        p.UpdatedAt_2.Time,
		},
		AvgRating:   *float64ToProtoDecimal(p.AvgRating),
		ReviewCount: p.ReviewCount,
	}
}

func (r *PropertyRepository) mapSQLCSearchResultToEntity(p db.SearchPropertiesWithDetailsRow) *entity.PropertyWithDetails {
	return &entity.PropertyWithDetails{
		Property: entity.Property{
			ID:          p.ID,
			Title:       p.Title,
			Description: toNullStringPtr(p.Description),
			Category:    entity.PropertyCategory(p.Category),
			Type:        entity.PropertyType(p.Type),
			Address:     p.Address,
			City:        p.City,
			State:       p.State,
			Country:     p.Country,
			ZipCode:     toNullStringPtr(p.ZipCode),
			OwnerID:     toNullUUIDPtr(p.OwnerID),
			Status:      entity.PropertyStatus(p.Status),
			CreatedAt:   toTime(p.CreatedAt),
			UpdatedAt:   toTime(p.UpdatedAt),
		},
		PropertyDetail: &entity.PropertyDetail{
			Bedrooms:      toNullInt32Ptr(p.Bedrooms),
			Bathrooms:     toNullInt32Ptr(p.Bathrooms),
			SquareFootage: toProtoDecimal(p.SquareFootage),
			LotSize:       toProtoDecimal(p.LotSize),
			YearBuilt:     toNullInt32Ptr(p.YearBuilt),
			Stories:       toNullInt32Ptr(p.Stories),
			GarageCount:   toNullInt32Ptr(p.GarageCount),
			HasBasement:   p.HasBasement.Bool,
			HasAttic:      p.HasAttic.Bool,
		},
		AvgRating:   *float64ToProtoDecimal(p.AvgRating),
		ReviewCount: p.ReviewCount,
	}
}

// --- conversion helpers ---

func toNullPropertyCategory(c *entity.PropertyCategory) db.NullPropertyCategory {
	if c == nil {
		return db.NullPropertyCategory{}
	}
	return db.NullPropertyCategory{PropertyCategory: db.PropertyCategory(*c), Valid: true}
}

func toNullPropertyType(t *entity.PropertyType) db.NullPropertyType {
	if t == nil {
		return db.NullPropertyType{}
	}
	return db.NullPropertyType{PropertyType: db.PropertyType(*t), Valid: true}
}

func toNullPropertyStatus(s *entity.PropertyStatus) db.NullPropertyStatus {
	if s == nil {
		return db.NullPropertyStatus{}
	}
	return db.NullPropertyStatus{PropertyStatus: db.PropertyStatus(*s), Valid: true}
}

func derefString(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}

func derefInt32(i *int32) int32 {
	if i != nil {
		return *i
	}
	return 0
}

func derefBool(b *bool) bool {
	if b != nil {
		return *b
	}
	return false
}

func derefPropertyCategory(c *entity.PropertyCategory) entity.PropertyCategory {
	if c != nil {
		return *c
	}
	return ""
}

func derefPropertyType(t *entity.PropertyType) entity.PropertyType {
	if t != nil {
		return *t
	}
	return ""
}

func derefPropertyStatus(s *entity.PropertyStatus) entity.PropertyStatus {
	if s != nil {
		return *s
	}
	return ""
}

func protoDecimalValue(d interface{ GetValue() string }) string {
	if d == nil {
		return ""
	}
	return d.GetValue()
}

func protoDecimalValueOrZero(d interface{ GetValue() string }) string {
	if d == nil {
		return "0"
	}
	v := d.GetValue()
	if v == "" {
		return "0"
	}
	return v
}
