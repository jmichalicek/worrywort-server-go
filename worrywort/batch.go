package worrywort

// Models and functions for brew batch management

import "time"

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

	recipeUrl string

	createdAt time.Time
	updatedAt time.Time

	createdBy user
}

func NewBatch(id int64, name string, brewedDate, bottledDate time.Time, volumeBoiled, volumeInFermenter float32,
	volumeUnits VolumeUnitType, createdBy user, createdAt, updatedAt time.Time, brewNotes, tastingNotes,
	recipeUrl string) batch {
	return batch{id: id, name: name, brewedDate: brewedDate, bottledDate: bottledDate, volumeBoiled: volumeBoiled,
		volumeInFermenter: volumeInFermenter, volumeUnits: volumeUnits, createdBy: createdBy, createdAt: createdAt,
		updatedAt: updatedAt, brewNotes: brewNotes, tastingNotes: tastingNotes, recipeUrl: recipeUrl}
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
