package propertyhandler

import (
	"context"
	"strconv"

	pb "github.com/demola234/property/infrastructure/api/grpc"
	"github.com/demola234/property/internal/domain/entity"
	"github.com/demola234/property/internal/usecases"

	"github.com/google/uuid"
	"google.golang.org/genproto/googleapis/type/decimal"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type PropertyHandler struct {
	propertyUsecase usecases.PropertyUsecase
	pb.UnimplementedPropertyServiceServer
}

func NewPropertyHandler(propertyUsecase usecases.PropertyUsecase) *PropertyHandler {
	return &PropertyHandler{
		propertyUsecase: propertyUsecase,
	}
}

func (p *PropertyHandler) CreateProperty(ctx context.Context, req *pb.CreatePropertyRequest) (*pb.CreatePropertyResponse, error) {
	userID, err := uuid.Parse(req.GetOwnerId())
	if err != nil {
		return nil, status.Errorf(400, "Invalid user ID")
	}

	desc := req.GetDescription()
	zip := req.GetZipCode()
	priceStr := strconv.FormatFloat(req.GetPrice(), 'f', -1, 64)
	ownerID := userID

	property := &entity.Property{
		Title:       req.GetTitle(),
		Description: &desc,
		Price:       decimal.Decimal{Value: priceStr},
		Type:        entity.PropertyType(req.GetType()),
		Address:     req.GetAddress(),
		ZipCode:     &zip,
		OwnerID:     &ownerID,
		Status:      entity.PropertyStatus(req.GetStatus()),
	}

	if err := p.propertyUsecase.CreateProperty(ctx, property); err != nil {
		return nil, status.Errorf(500, "Unable to create property: %v", err)
	}

	return &pb.CreatePropertyResponse{
		Id: property.ID.String(),
	}, nil
}

func (p *PropertyHandler) GetProperties(ctx context.Context, req *pb.GetPropertiesRequest) (*pb.GetPropertiesResponse, error) {
	properties, err := p.propertyUsecase.GetProperties(ctx, req.GetLimit(), req.GetOffset())
	if err != nil {
		return nil, status.Errorf(500, "Unable to get properties: %v", err)
	}

	propertyResponses := make([]*pb.Property, len(properties))
	for i, property := range properties {
		propertyResponses[i] = mapEntityToProto(property)
	}

	return &pb.GetPropertiesResponse{
		Properties: propertyResponses,
	}, nil
}

func (p *PropertyHandler) GetPropertiesByOwner(ctx context.Context, req *pb.GetPropertiesByOwnerRequest) (*pb.GetPropertiesByOwnerResponse, error) {
	uuidUser, err := uuid.Parse(req.GetOwnerId())
	if err != nil {
		return nil, status.Errorf(400, "invalid owner ID: %v", err)
	}

	properties, err := p.propertyUsecase.GetPropertiesByOwner(ctx, uuid.NullUUID{UUID: uuidUser, Valid: true}, req.GetLimit(), req.GetOffset())
	if err != nil {
		return nil, status.Errorf(500, "Unable to get properties: %v", err)
	}

	propertyResponses := make([]*pb.Property, len(properties))
	for i, property := range properties {
		propertyResponses[i] = mapEntityToProto(property)
	}

	return &pb.GetPropertiesByOwnerResponse{
		Properties: propertyResponses,
	}, nil
}

func (p *PropertyHandler) GetPropertyByID(ctx context.Context, req *pb.GetPropertyByIDRequest) (*pb.GetPropertyByIDResponse, error) {
	uuidProperty, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Errorf(400, "invalid property ID: %v", err)
	}

	property, err := p.propertyUsecase.GetPropertyByID(ctx, uuidProperty)
	if err != nil {
		return nil, status.Errorf(500, "Unable to get property: %v", err)
	}

	return &pb.GetPropertyByIDResponse{
		Property: mapEntityToProto(property),
	}, nil
}

func (p *PropertyHandler) UpdateProperty(ctx context.Context, req *pb.UpdatePropertyRequest) (*pb.UpdatePropertyResponse, error) {
	uuidProperty, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Errorf(400, "invalid property ID: %v", err)
	}

	userProperty, err := p.propertyUsecase.GetPropertyByID(ctx, uuidProperty)
	if err != nil {
		return nil, status.Errorf(500, "Unable to get property: %v", err)
	}

	if userProperty.OwnerID == nil || userProperty.OwnerID.String() != req.GetOwnerId() {
		return nil, status.Errorf(403, "You are not authorized to update this property")
	}

	desc := req.GetDescription()
	zip := req.GetZipCode()
	priceStr := strconv.FormatFloat(req.GetPrice(), 'f', -1, 64)

	property := &entity.Property{
		ID:          uuidProperty,
		Title:       req.GetTitle(),
		Description: &desc,
		Price:       decimal.Decimal{Value: priceStr},
		Type:        entity.PropertyType(req.GetType()),
		Address:     req.GetAddress(),
		ZipCode:     &zip,
		Status:      entity.PropertyStatus(req.GetStatus()),
	}

	if err := p.propertyUsecase.UpdateProperty(ctx, property); err != nil {
		return nil, err
	}

	return &pb.UpdatePropertyResponse{}, nil
}

func (p *PropertyHandler) DeleteProperty(ctx context.Context, req *pb.DeletePropertyRequest) (*pb.DeletePropertyResponse, error) {
	uuidProperty, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Errorf(400, "invalid property ID: %v", err)
	}

	userProperty, err := p.propertyUsecase.GetPropertyByID(ctx, uuidProperty)
	if err != nil {
		return nil, status.Errorf(500, "Unable to get property: %v", err)
	}

	if userProperty.OwnerID == nil || userProperty.OwnerID.String() != req.GetOwnerId() {
		return nil, status.Errorf(403, "You are not authorized to delete this property")
	}

	if err := p.propertyUsecase.DeleteProperty(ctx, uuidProperty); err != nil {
		return nil, err
	}

	return &pb.DeletePropertyResponse{}, nil
}

func mapEntityToProto(property *entity.Property) *pb.Property {
	var price float64
	if property.Price.Value != "" {
		price, _ = strconv.ParseFloat(property.Price.Value, 64)
	}

	var ownerID string
	if property.OwnerID != nil {
		ownerID = property.OwnerID.String()
	}

	var description string
	if property.Description != nil {
		description = *property.Description
	}

	var zipCode string
	if property.ZipCode != nil {
		zipCode = *property.ZipCode
	}

	return &pb.Property{
		Id:          property.ID.String(),
		Title:       property.Title,
		Description: description,
		Price:       price,
		Type:        string(property.Type),
		Address:     property.Address,
		ZipCode:     zipCode,
		OwnerId:     ownerID,
		Status:      string(property.Status),
		CreatedAt:   timestamppb.New(property.CreatedAt),
		UpdatedAt:   timestamppb.New(property.UpdatedAt),
	}
}
