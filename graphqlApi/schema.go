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
		batchSensorAssociations(first: Int after: String batchId: ID sensorId: ID): BatchSensorAssociationConnection!
		# given sensors like iSpindel and Tilt, perhaps just "sensor" with a type
		# is more appropriate?
		sensor(id: ID!): Sensor
		sensors(first: Int after: String): SensorConnection!
		temperatureMeasurement(id: ID!): TemperatureMeasurement
		temperatureMeasurements(first: Int after: String sensorId: ID batchId: ID): TemperatureMeasurementConnection
	}

	type Mutation {
		associateSensorToBatch(input: AssociateSensorToBatchInput!): AssociateSensorToBatchPayload
		login(username: String!, password: String!): AuthToken
		# Might remove createTemperatureMeasurement in favor of having those created via more IoT Friendly
		# system such as mqtt.  Then system can look at relationships to attach to batch, fermenter, etc.
		# but will definitely need an updateTemperatureMeasurement() to edit - ie. attach to a batch later, etc.
		createTemperatureMeasurement(input: CreateTemperatureMeasurementInput!): CreateTemperatureMeasurementPayload
		createBatch(input: CreateBatchInput!): CreateBatchPayload
		createSensor(input: CreateSensorInput!): CreateSensorPayload
		updateBatchSensorAssociation(input: UpdateBatchSensorAssociationInput!): UpdateBatchSensorAssociationPayload
	}

	enum VolumeUnit {
		GALLON
		QUART
	}

	enum TemperatureUnit {
		FAHRENHEIT
		CELSIUS
	}

	# RFC3339 formatted DateTime
	scalar DateTime

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
		brewedDate: DateTime
		bottledDate: DateTime
		volumeBoiled: Float
		volumeInFermentor: Float
		volumeUnits: VolumeUnit!
		originalGravity: Float
		finalGravity: Float
		recipeURL: String!
		createdAt: DateTime!
		updatedAt: DateTime!
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

	type BatchSensorAssociation {
		# datetime
		associatedAt: DateTime!
		batch: Batch!
		description: String
		disassociatedAt: DateTime
		id: ID!
		sensor: Sensor!
	}

	type BatchSensorAssociationConnection {
		pageInfo: PageInfo!
		edges: [BatchSensorAssociationEdge!]
	}

	type BatchSensorAssociationEdge {
		cursor: String!
		node: BatchSensorAssociation!
	}

	type AssociateSensorToBatchPayload {
		batchSensorAssociation: BatchSensorAssociation
	}

	type UpdateBatchSensorAssociationPayload {
		batchSensorAssociation: BatchSensorAssociation
	}

	type CreateBatchPayload {
		batch: Batch
	}

	type CreateSensorPayload {
		sensor: Sensor
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
		recordedAt: DateTime!
		# The batch being monitored, if this was actively monitoring a batch
		batch: Batch
		# The Sensor which took the measurement
		sensor: Sensor
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
		createdAt: DateTime!
		updatedAt: DateTime!
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
		createdAt: DateTime!
		updatedAt: DateTime!
	}

	type UserError {
		field: [String!]
		error: String!
	}

	# Input data to create a Batch
	input CreateBatchInput {
		# A name for the Batch
		name: String!
		# The date and time the batch was brewed
		brewedAt: DateTime!
		bottledAt: DateTime
		volumeBoiled: Float
		volumeInFermentor: Float
		volumeUnits: String
		originalGravity: Float
		finalGravity: Float
		// MaxTemperature     *float64
		// MinTemperature     *float64
		// AverageTemperature *float64  not even sure this should be on the model...
		recipeURL: String
		tastingNotes: String
	}

	# Input data to create a sensor
	input CreateSensorInput {
		# A useful name for the sensor
		name: String!
	}

	# Input data to create a TemperatureMeasurement
	input CreateTemperatureMeasurementInput {
		# The temperature taken
		temperature: Float!
		# The date and time the temperature was recorded by the sensor
		recordedAt: DateTime!
		# The id of the Sensor which took the measurement
		sensorId: ID!
		# The units the temperature was taken in
		units: TemperatureUnit!
	}

	# Input data to associate a Sensor to a Batch
	input AssociateSensorToBatchInput {
		batchId: ID!
		description: String
		sensorId: ID!
	}

	# Update a batchSensorAssociation to match the given input
	input UpdateBatchSensorAssociationInput {
		associatedAt: DateTime!
		id: ID!
		description: String
		disassociatedAt: DateTime
	}
	`

// TODO: Make a DateTime type for the various datetimes
