package graphqlApi


// fermenter(id: ID!): Fermenter
// thermometer(id: ID!): Thermometer
// temperatureMeasurement(id: ID!): TemperatureMeasurement

// part of schema section
// mutation: Mutation
//Below type Query once there is a Mutation
// type Mutation {}
var Schema = `
	schema {
		query: Query

	}

	type Query {
		currentUser(): User
		batch(id: ID!): Batch

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

	type User {
		id: ID!
		firstName: String!
		lastName: String!
		email: String!
		createdAt: String
		updatedAt: String
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
		createdAt: String
		updatedAt: String
		createdBy: User
	}
	`
