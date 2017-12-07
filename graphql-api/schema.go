package graphqlApi

import graphql "github.com/neelance/graphql-go"

var Schema = `
	schema {
		query: Query
		mutation: Mutation
	}

	type Query {
		currentUser(): User
		batch(id: ID!): Batch
		fermenter(id: ID!): Fermenter
		thermometer(id: ID!): Thermometer
		temperatureMeasurement(id: ID!): TemperatureMeasurement
	}

	type Mutation {}

	enum VolumeUnit {
		GALLON
		QUART
	}

	enum TemperatureUnit {
		FAHRENHEIT
		CELSIUS
	}

	enum FermenterStyle {
		BUCKET
		CARBOY
		CONICAL
	}

	type User {
		id: ID!
		firstName: String
		lastName: String
		email: String
		createdAt: String
		updatedAt: String
	}

	type Batch {
		id: ID!
		name: String
		brewNotes: String
		tastingNotes: String
		brewedDate: String
		bottledDate: String
		volumeBoiled: Float
		volumeInFermenter: Float
		volumeUnits: VolumeUnit
		originalGravity: Float
		finalGravity: Float
		recipeURL: String
		createdAt: String
		updatedAt: String
		createdBy: User
	}
	`
