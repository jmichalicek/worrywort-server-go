package worrywort;
// Models and functions for brew batch management

import "time";

type VolumeUnitType int32
const (
    GALLON VolumeType = iota
    QUART VolumeType
)

type TemperatureUnitType int32
const (
    FAHRENHEIT TemperatureUnitType = iota
    CELSIUS TemperatureUnitType
)


// Should these be exportable if I am going to use factory methods?  NewBatch() etc?
// as long as I provide a Batcher interface or whatever?
type batch struct {
  name string
  brew_notes string
  tasting_notes string
  brewed_date time.Date
  bottled_date time.Date

  volume_boiled float32;
  volume_in_fermenter float32
  volume_units VolumeUnitType

  original_gravity float32
  final_gravity float32

  recipe_url string

  created_at time.Date
  updated_at time.Date

  created_by user;
}

type fermenter struct {
    name string;
    description string;

}
type thermometer struct {}
