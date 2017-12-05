package worrywort

// Models and functions for brew batch management

import (
	// "net/url"
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

// Should these be exportable if I am going to use factory methods?  NewBatch() etc?
// as long as I provide a Batcher interface or whatever?
type Batch struct {
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

	// handle this as a string.  It makes nearly everything easier and can easily be run through
	// url.Parse if needed
	recipeURL string

	createdAt time.Time
	updatedAt time.Time

	createdBy User
}

func (b Batch) Id() int64                   { return b.id }
func (b Batch) Name() string                { return b.name }
func (b Batch) BrewNotes() string           { return b.brewNotes }
func (b Batch) TastingNotes() string        { return b.tastingNotes }
func (b Batch) BrewedDate() time.Time       { return b.brewedDate }
func (b Batch) BottledDate() time.Time      { return b.bottledDate }
func (b Batch) VolumeBoiled() float32       { return b.volumeBoiled }
func (b Batch) VolumeInFermenter() float32  { return b.volumeInFermenter }
func (b Batch) VolumeUnits() VolumeUnitType { return b.volumeUnits }
func (b Batch) OriginalGravity() float32    { return b.originalGravity }
func (b Batch) FinalGravity() float32       { return b.finalGravity }
func (b Batch) RecipeURL() string          { return b.recipeURL }  // this could even return a parsed URL object...
func (b Batch) CreatedAt() time.Time        { return b.createdAt }
func (b Batch) UpdatedAt() time.Time        { return b.updatedAt }
func (b Batch) CreatedBy() User             { return b.createdBy }

func NewBatch(id int64, name string, brewedDate, bottledDate time.Time, volumeBoiled, volumeInFermenter float32,
	volumeUnits VolumeUnitType, originalGravity, finalGravity float32, createdBy User, createdAt, updatedAt time.Time,
	brewNotes, tastingNotes string, recipeURL string) Batch {
	return Batch{id: id, name: name, brewedDate: brewedDate, bottledDate: bottledDate, volumeBoiled: volumeBoiled,
		volumeInFermenter: volumeInFermenter, volumeUnits: volumeUnits, createdBy: createdBy, createdAt: createdAt,
		updatedAt: updatedAt, brewNotes: brewNotes, tastingNotes: tastingNotes, recipeURL: recipeURL,
		originalGravity: originalGravity, finalGravity: finalGravity}
}

type Fermenter struct {
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
	createdBy     User

	createdAt time.Time
	updatedAt time.Time
}

func NewFermenter(id int64, name, description string, volume float32, volumeUnits VolumeUnitType,
	fermenterType FermenterStyleType, isActive, isAvailable bool, createdBy User, createdAt, updatedAt time.Time) Fermenter {
	return Fermenter{id: id, name: name, description: description, volume: volume, volumeUnits: volumeUnits,
		fermenterType: fermenterType, isActive: isActive, isAvailable: isAvailable, createdBy: createdBy,
		createdAt: createdAt, updatedAt: updatedAt}
}

// possibly should live elsewhere

// Thermometer will need some other unique identifier which the unit itself
// can know, ideally.
// TODO: This may also want extra metadata such as model or type?  That is probably
// going too far for now, so keep it simple.
type Thermometer struct {
	id        int64
	name      string
	createdBy User

	createdAt time.Time
	updatedAt time.Time
}

// Returns a new Thermometer
func NewThermometer(id int64, name string, createdBy User, createdAt, updatedAt time.Time) Thermometer {
	return Thermometer{id: id, name: name, createdBy: createdBy, createdAt: createdAt, updatedAt: updatedAt}
}

// A single recorded temperature measurement from a thermometer
// This may get some tweaking to play nicely with data stored in Postgres or Influxdb
type TemperatureMeasurement struct {
	id           string // use a uuid
	temperature  float64
	units        TemperatureUnitType
	timeRecorded time.Time // when the measurement was recorded
	batch        Batch
	thermometer  Thermometer
	fermenter    Fermenter

	// not sure createdBy is a useful name in this case vs just `user` but its consistent
	createdBy User

	// when the record was created
	createdAt time.Time
	updatedAt time.Time
}

func NewTemperatureMeasurement(id string, temperature float64, units TemperatureUnitType, batch Batch,
	thermometer Thermometer, fermenter Fermenter, timeRecorded, createdAt, updatedAt time.Time, createdBy User) TemperatureMeasurement {
	return TemperatureMeasurement{id: id, temperature: temperature, units: units, batch: batch, thermometer: thermometer,
		fermenter: fermenter, timeRecorded: timeRecorded, createdAt: createdAt, updatedAt: updatedAt, createdBy: createdBy}
}
