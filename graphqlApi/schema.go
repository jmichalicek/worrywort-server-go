package graphqlApi

// fermenter(id: ID!): Fermenter
// temperatureSensor(id: ID!): TemperatureSensor
// temperatureMeasurement(id: ID!): TemperatureMeasurement

// part of schema section
// mutation: Mutation
//Below type Query once there is a Mutation
// type Mutation {}
var Schema = `
	schema {
		query: Query
		mutation: Mutation
	}

	type Query {
		currentUser(): User
		batch(id: ID!): Batch
		batches(): [Batch]
	}

	type Mutation {
		login(username: String!, password: String!): AuthToken
	}

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

	type AuthToken {
		id: ID!
		token: String!
	}

	type User {
		id: ID!
		firstName: String!
		lastName: String!
		email: String!
		createdAt: String!
		updatedAt: String!
	}

	type Batch {
		id: ID!
		name: String!
		brewNotes: String!
		tastingNotes: String!
		brewedDate: String
		bottledDate: String
		volumeBoiled: Float
		volumeInFermenter: Float
		volumeUnits: VolumeUnit!
		originalGravity: Float
		finalGravity: Float
		recipeURL: String!
		createdAt: String!
		updatedAt: String!
		createdBy: User
	}

	type Fermenter {
		id: ID!
	}

	type TemperatureSensor {
		id: ID!
		name: String
		createdBy: User
	}

	type TemperatureMeasurement {
		id: ID!
		temperature: Float!
		units: TemperatureUnit!
		recordedAt: String!
		Batch: Batch
		temperatureSensor: TemperatureSensor!
		fermenter: Fermenter
	}
	`

// TODO: Make a DateTime type for the various datetimes
