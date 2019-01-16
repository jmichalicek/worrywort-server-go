package graphqlApi

// fermentor(id: ID!): Fermentor
// sensor(id: ID!): Sensor
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
		batches(first: Int after: String): BatchConnection!
		# given sensors like iSpindel and Tilt, perhaps just "sensor" with a type
		# is more appropriate?
		sensor(id: ID!): Sensor
		sensors(first: Int after: String): SensorConnection!
		temperatureMeasurement(id: ID!): TemperatureMeasurement
		temperatureMeasurements(first: Int after: String sensorId: ID batchId: ID): TemperatureMeasurementConnection
	}

	type Mutation {
		# Currently broken because I made the whole /graphql endpoint require a token for now
		login(username: String!, password: String!): AuthToken
		# Might remove createTemperatureMeasurement in favor of having those created via more IoT Friendly
		# system such as mqtt.  Then system can look at relationships to attach to batch, fermenter, etc.
		# but will definitely need an updateTemperatureMeasurement() to edit - ie. attach to a batch later, etc.
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

	enum FermentorStyle {
		BUCKET
		CARBOY
		CONICAL
	}

	type AuthToken {
		id: ID!
		token: String!
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
		volumeInFermentor: Float
		volumeUnits: VolumeUnit!
		originalGravity: Float
		finalGravity: Float
		recipeURL: String!
		createdAt: String!
		updatedAt: String!
		createdBy: User
	}

	type BatchConnection {
		pageInfo: PageInfo!
		edges: [BatchEdge!]
	}

	type BatchEdge {
		cursor: String!
		node: Batch!
	}

	type CreateTemperatureMeasurementPayload {
		temperatureMeasurement: TemperatureMeasurement
	}

	type Fermentor {
		id: ID!
	}

	# A measurement taken by a Sensor
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
		# The Sensor which took the measurement
		sensor: Sensor
		# The Fermentor the sensor was attached to, if any
		fermentor: Fermentor
	}

	type TemperatureMeasurementConnection {
		pageInfo: PageInfo!
		edges: [TemperatureMeasurementEdge!]
	}

	type TemperatureMeasurementEdge {
		cursor: String!
		node: TemperatureMeasurement!
	}

	type Sensor {
		id: ID!
		# Friendly name of the temperature sensor
		name: String!
		createdBy: User
	}

	type SensorConnection {
		pageInfo: PageInfo!
		edges: [SensorEdge!]
	}

	type SensorEdge {
		cursor: String!
		node: Sensor!
	}

	type PageInfo {
		hasPreviousPage: Boolean!
		hasNextPage: Boolean!
	}

	type User {
		id: ID!
		firstName: String!
		lastName: String!
		email: String!
		createdAt: String!
		updatedAt: String!
	}

	# Input data to create a TemperatureMeasurement
	input CreateTemperatureMeasurementInput {
		# The temperature taken
		temperature: Float!
		# The date and time the temperature was recorded by the sensor
		recordedAt: String!
		# The id of the Sensor which took the measurement
		sensorId: ID!
		# The units the temperature was taken in
		units: TemperatureUnit!
		# The Batch being monitored if this was monitoring a Batch
		batchId: ID
	}
	`

// TODO: Make a DateTime type for the various datetimes
