package worrywort

// Models and functions for brew batch management

import (
	"net/url"
	"time"
)

type VolumeUnitType int32

const (
	GALLON VolumeUnitType = iota
	QUART
)

type TemperatureUnitType int32

const (
	FAHRENHEIT TemperatureUnitType = iota
	CELSIUS
)

type FermenterStyleType int32

const (
	BUCKET FermenterStyleType = iota
	CARBOY
	CONICAL
)

type Batcher interface {
	Id() int64
	Name() string
	BrewNotes() string
	TastingNotes() string
	BrewedDate() time.Time
	BottledDate() time.Time
	VolumeBoiled() float32
	VolumeInFermenter() float32
	VolumeUnits() VolumeUnitType
	OriginalGravity() float32
	FinalGravity() float32
	RecipeURL() url.URL
	CreatedAt() time.Time
	UpdatedAt() time.Time
	CreatedBy() user
}

// Should these be exportable if I am going to use factory methods?  NewBatch() etc?
// as long as I provide a Batcher interface or whatever?
type batch struct {
	id           int64
	name         string
	brewNotes    string
	tastingNotes string
	brewedDate   time.Time
	bottledDate  time.Time

	volumeBoiled      float32
	volumeInFermenter float32
	volumeUnits       VolumeUnitType

	originalGravity float32
	finalGravity    float32

	recipeURL url.URL

	createdAt time.Time
	updatedAt time.Time

	createdBy user
}

func (b batch) Id() int64 { return b.id }
func (b batch) Name() string { return b.name }
func (b batch) BrewNotes() string { return b.brewNotes }
func (b batch) TastingNotes() string { return b.tastingNotes }
func (b batch) BrewedDate() time.Time { return b.brewedDate }
func (b batch) BottledDate() time.Time { return b.bottledDate }
func (b batch) VolumeBoiled() float32 { return b.volumeBoiled }
func (b batch) VolumeInFermenter() float32 { return b.volumeInFermenter }
func (b batch) VolumeUnits() VolumeUnitType { return b.volumeUnits }
func (b batch) OriginalGravity() float32 { return b.originalGravity }
func (b batch) FinalGravity() float32 { return b.finalGravity }
func (b batch) RecipeURL() url.URL { return b.recipeURL }
func (b batch) CreatedAt() time.Time { return b.createdAt }
func (b batch) UpdatedAt() time.Time { return b.updatedAt }
func (b batch) CreatedBy() user { return b.createdBy }

func NewBatch(id int64, name string, brewedDate, bottledDate time.Time, volumeBoiled, volumeInFermenter float32,
	volumeUnits VolumeUnitType, originalGravity, finalGravity float32, createdBy user, createdAt, updatedAt time.Time,
	brewNotes, tastingNotes string, recipeURL url.URL) batch {
	return batch{id: id, name: name, brewedDate: brewedDate, bottledDate: bottledDate, volumeBoiled: volumeBoiled,
		volumeInFermenter: volumeInFermenter, volumeUnits: volumeUnits, createdBy: createdBy, createdAt: createdAt,
		updatedAt: updatedAt, brewNotes: brewNotes, tastingNotes: tastingNotes, recipeURL: recipeURL,
		originalGravity: originalGravity, finalGravity: finalGravity}
}

type fermenter struct {
	// I could use name + user composite key for pk on these in the db, but I'm probably going to be lazy
	// and take the standard ORM-ish route and use an int or uuid  Int for now.
	id            int64
	name          string
	description   string
	volume        float32
	volumeUnits   VolumeUnitType
	fermenterType FermenterStyleType
	isActive      bool
	isAvailable   bool
	createdBy     user
}

func NewFermenter(id int64, name, description string, volume float32, volumeUnits VolumeUnitType,
	fermenterType FermenterStyleType, isActive, isAvailable bool, createdBy user) fermenter {
	return fermenter{id: id, name: name, description: description, volume: volume, volumeUnits: volumeUnits,
		fermenterType: fermenterType, isActive: isActive, isAvailable: isAvailable, createdBy: createdBy}
}

type thermometer struct{}
