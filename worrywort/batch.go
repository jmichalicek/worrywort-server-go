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
type batch struct {
	ID           int64
	Name         string
	BrewNotes    string
	TastingNotes string
	BrewedDate   time.Time
	BottledDate  time.Time
	VolumeBoiled      float64
	VolumeInFermenter float64
	VolumeUnits       VolumeUnitType
	OriginalGravity float64
	FinalGravity    float64

	// handle this as a string.  It makes nearly everything easier and can easily be run through
	// url.Parse if needed
	RecipeURL string

	CreatedAt time.Time
	UpdatedAt time.Time

	CreatedBy User
}

type Batch struct {
	batch
}

func (b Batch) ID() int64                   { return b.batch.ID }
func (b Batch) Name() string                { return b.batch.Name }
func (b Batch) BrewNotes() string           { return b.batch.BrewNotes }
func (b Batch) TastingNotes() string        { return b.batch.TastingNotes }
func (b Batch) BrewedDate() time.Time       { return b.batch.BrewedDate }
func (b Batch) BottledDate() time.Time      { return b.batch.BottledDate }
func (b Batch) VolumeBoiled() float64       { return b.batch.VolumeBoiled }
func (b Batch) VolumeInFermenter() float64  { return b.batch.VolumeInFermenter }
func (b Batch) VolumeUnits() VolumeUnitType { return b.batch.VolumeUnits }
func (b Batch) OriginalGravity() float64    { return b.batch.OriginalGravity }
func (b Batch) FinalGravity() float64       { return b.batch.FinalGravity }
func (b Batch) RecipeURL() string           { return b.batch.RecipeURL } // this could even return a parsed URL object...
func (b Batch) CreatedAt() time.Time        { return b.batch.CreatedAt }
func (b Batch) UpdatedAt() time.Time        { return b.batch.UpdatedAt }
func (b Batch) CreatedBy() User             { return b.batch.CreatedBy }

func NewBatch(id int64, name string, brewedDate, bottledDate time.Time, volumeBoiled, volumeInFermenter float64,
	volumeUnits VolumeUnitType, originalGravity, finalGravity float64, createdBy User, createdAt, updatedAt time.Time,
	brewNotes, tastingNotes string, recipeURL string) Batch {
	return Batch{batch: batch{ID: id, Name: name, BrewedDate: brewedDate, BottledDate: bottledDate, VolumeBoiled: volumeBoiled,
		VolumeInFermenter: volumeInFermenter, VolumeUnits: volumeUnits, CreatedBy: createdBy, CreatedAt: createdAt,
		UpdatedAt: updatedAt, BrewNotes: brewNotes, TastingNotes: tastingNotes, RecipeURL: recipeURL,
		OriginalGravity: originalGravity, FinalGravity: finalGravity}}
}

type fermenter struct {
	// I could use name + user composite key for pk on these in the db, but I'm probably going to be lazy
	// and take the standard ORM-ish route and use an int or uuid  Int for now.
	ID            int64
	Name          string
	Description   string
	Volume        float64
	VolumeUnits   VolumeUnitType
	FermenterType FermenterStyleType
	IsActive      bool
	IsAvailable   bool
	CreatedBy     User

	CreatedAt time.Time
	UpdatedAt time.Time
}

type Fermenter struct {
	fermenter
}

func (f Fermenter) ID() int64                         { return f.fermenter.ID }
func (f Fermenter) Name() string                      { return f.fermenter.Name }
func (f Fermenter) Description() string               { return f.fermenter.Description }
func (f Fermenter) VolumeUnits() VolumeUnitType       { return f.fermenter.VolumeUnits }
func (f Fermenter) FermenterType() FermenterStyleType { return f.fermenter.FermenterType }
func (f Fermenter) IsActive() bool                    { return f.fermenter.IsActive }
func (f Fermenter) IsAvailable() bool                 { return f.fermenter.IsAvailable }
func (f Fermenter) CreatedBy() User                   { return f.fermenter.CreatedBy }
func (f Fermenter) CreatedAt() time.Time              { return f.fermenter.CreatedAt }
func (f Fermenter) UpdatedAt() time.Time              { return f.fermenter.UpdatedAt }

func NewFermenter(id int64, name, description string, volume float64, volumeUnits VolumeUnitType,
	fermenterType FermenterStyleType, isActive, isAvailable bool, createdBy User, createdAt, updatedAt time.Time) Fermenter {
	return Fermenter{fermenter{ID: id, Name: name, Description: description, Volume: volume, VolumeUnits: volumeUnits,
		FermenterType: fermenterType, IsActive: isActive, IsAvailable: isAvailable, CreatedBy: createdBy,
		CreatedAt: createdAt, UpdatedAt: updatedAt}}
}

// possibly should live elsewhere

// Thermometer will need some other unique identifier which the unit itself
// can know, ideally.
// TODO: This may also want extra metadata such as model or type?  That is probably
// going too far for now, so keep it simple.
type thermometer struct {
	ID        int64
	Name      string
	CreatedBy User

	CreatedAt time.Time
	UpdatedAt time.Time
}

type Thermometer struct {
	thermometer
}

func (t Thermometer) ID() int64            { return t.thermometer.ID }
func (t Thermometer) Name() string         { return t.thermometer.Name }
func (t Thermometer) CreatedBy() User      { return t.thermometer.CreatedBy }
func (t Thermometer) CreatedAt() time.Time { return t.thermometer.CreatedAt }
func (t Thermometer) UpdatedAt() time.Time { return t.thermometer.UpdatedAt }

// Returns a new Thermometer
func NewThermometer(id int64, name string, createdBy User, createdAt, updatedAt time.Time) Thermometer {
	return Thermometer{thermometer{ID: id, Name: name, CreatedBy: createdBy, CreatedAt: createdAt, UpdatedAt: updatedAt}}
}

// A single recorded temperature measurement from a thermometer
// This may get some tweaking to play nicely with data stored in Postgres or Influxdb
type temperatureMeasurement struct {
	ID           string // use a uuid
	Temperature  float64
	Units        TemperatureUnitType
	TimeRecorded time.Time // when the measurement was recorded
	Batch        Batch
	Thermometer  Thermometer
	Fermenter    Fermenter

	// not sure createdBy is a useful name in this case vs just `user` but its consistent
	CreatedBy User

	// when the record was created
	CreatedAt time.Time
	UpdatedAt time.Time
}

type TemperatureMeasurement struct {
	temperatureMeasurement
}

func (t TemperatureMeasurement) ID() string                 { return t.temperatureMeasurement.ID }
func (t TemperatureMeasurement) Temperature() float64       { return t.temperatureMeasurement.Temperature }
func (t TemperatureMeasurement) Units() TemperatureUnitType { return t.temperatureMeasurement.Units }
func (t TemperatureMeasurement) TimeRecorded() time.Time    { return t.temperatureMeasurement.TimeRecorded }
func (t TemperatureMeasurement) Batch() Batch               { return t.temperatureMeasurement.Batch }
func (t TemperatureMeasurement) Thermometer() Thermometer   { return t.temperatureMeasurement.Thermometer }
func (t TemperatureMeasurement) Fermenter() Fermenter       { return t.temperatureMeasurement.Fermenter }
func (t TemperatureMeasurement) CreatedBy() User            { return t.temperatureMeasurement.CreatedBy }
func (t TemperatureMeasurement) CreatedAt() time.Time       { return t.temperatureMeasurement.CreatedAt }
func (t TemperatureMeasurement) UpdatedAt() time.Time       { return t.temperatureMeasurement.UpdatedAt }

func NewTemperatureMeasurement(id string, temperature float64, units TemperatureUnitType, batch Batch,
	thermometer Thermometer, fermenter Fermenter, timeRecorded, createdAt, updatedAt time.Time, createdBy User) TemperatureMeasurement {
	return TemperatureMeasurement{temperatureMeasurement{ID: id, Temperature: temperature, Units: units, Batch: batch,
		Thermometer: thermometer, Fermenter: fermenter, TimeRecorded: timeRecorded, CreatedAt: createdAt,
		UpdatedAt: updatedAt, CreatedBy: createdBy}}
}
