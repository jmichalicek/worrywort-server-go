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
		# Returns a Batch by id for the currently authenticated user
		batch(id: ID!): Batch
		# Returns a list of batches for the currently authenticated user.
		batches(): [Batch!]
	}

	type Mutation {
		login(username: String!, password: String!): AuthToken
		createTemperatureMeasurement(input: CreateTemperatureMeasurementInput): CreateTemperatureMeasurementPayload
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
		# A name for the batch brewed
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
		# Friendly name of the temperature sensor
		name: String!
		createdBy: User
	}

	# Words
	type TemperatureMeasurement {
		id: ID!
		# The recorded temperature
		temperature: Float!
		# The units the temperature is recorded in
		units: TemperatureUnit!
		# The date and time the temperature was taken by the sensor
		recordedAt: String!
		# The batch being monitored, if this was actively monitoring a batch
		batch: Batch
		# The TemperatureSensor which took the measurement
		temperatureSensor: TemperatureSensor
		# The Fermenter the sensor was attached to, if any
		fermenter: Fermenter
	}

	type CreateTemperatureMeasurementPayload {
		temperatureMeasurement: TemperatureMeasurement
	}

	input CreateTemperatureMeasurementInput {
		# The temperature taken
		temperature: Float!
		# The date and time the temperature was recorded by the sensor
		recordedAt: String!
		# The id of the TemperatureSensor which took the measurement
		temperatureSensorId: ID!
		# The units the temperature was taken in
		units: TemperatureUnit!
		# The Batch being monitored if this was monitoring a Batch
		batchId: ID
	}
	`

// TODO: Make a DateTime type for the various datetimes
